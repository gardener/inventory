// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"

	compute "cloud.google.com/go/compute/apiv1"
	computepb "cloud.google.com/go/compute/apiv1/computepb"
	"github.com/hibiken/asynq"
	"google.golang.org/api/iterator"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/constants"
	"github.com/gardener/inventory/pkg/gcp/models"
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// TaskCollectTargetPools is the name of the task for collecting GCP
// Target Pools.
//
// For more information about Target Pools, please refer to the
// [Target Pools] documentation.
//
// [Target Pools]: https://cloud.google.com/load-balancing/docs/target-pools
const TaskCollectTargetPools = "gcp:task:collect-target-pools"

// CollectTargetPoolsPayload is the payload used for collecting GCP Target Pools
// for a given project.
type CollectTargetPoolsPayload struct {
	// ProjectID specifies the globally unique project id from which to
	// collect resources.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectTargetPoolsTask creates a new [asynq.Task] for collecting GCP
// Target Pools, without specifying a payload.
func NewCollectTargetPoolsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectTargetPools, nil)
}

// HandlCollectTargetPools is the handler, which collects GCP Target Pools.
func HandleCollectTargetPools(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting resources from all registered projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectTargetPools(ctx)
	}

	var payload CollectTargetPoolsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectTargetPools(ctx, payload)
}

// enqueueCollectTargetPools enqueues tasks for collecting GCP Target Pools from
// all known projects.
func enqueueCollectTargetPools(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if gcpclients.TargetPoolsClientset.Length() == 0 {
		logger.Warn("no GCP target pools clients found")
		return nil
	}

	// Enqueue tasks for all registered GCP Projects
	err := gcpclients.TargetPoolsClientset.Range(func(projectID string, _ *gcpclients.Client[*compute.TargetPoolsClient]) error {
		payload := CollectTargetPoolsPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP Target Pools",
				"project", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}
		task := asynq.NewTask(TaskCollectTargetPools, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"project", projectID,
		)

		return nil
	})

	return err
}

// collectTargetPools collects the GCP Target Pools from the project
// specified in the payload.
func collectTargetPools(ctx context.Context, payload CollectTargetPoolsPayload) error {
	client, ok := gcpclients.TargetPoolsClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting GCP target pools", "project", payload.ProjectID)

	pageSize := uint32(constants.PageSize)
	partialSuccess := true
	req := &computepb.AggregatedListTargetPoolsRequest{
		Project:              gcputils.ProjectFQN(payload.ProjectID),
		MaxResults:           &pageSize,
		ReturnPartialSuccess: &partialSuccess,
	}

	targetPools := make([]models.TargetPool, 0)
	targetPoolInstances := make([]models.TargetPoolInstance, 0)
	it := client.Client.AggregatedList(ctx, req)
	for {
		// The iterator returns a k/v pair, where the key represents a
		// specific GCP Region and the value is the slice of target
		// pools.
		pair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			logger.Error(
				"failed to get GCP Target Pools",
				"project", payload.ProjectID,
				"reason", err,
			)
			return err
		}

		region := gcputils.UnqualifyRegion(pair.Key)
		for _, tp := range pair.Value.TargetPools {
			// Target Pool
			item := models.TargetPool{
				TargetPoolID:      tp.GetId(),
				ProjectID:         payload.ProjectID,
				Name:              tp.GetName(),
				Description:       tp.GetDescription(),
				BackupPool:        tp.GetBackupPool(),
				CreationTimestamp: tp.GetCreationTimestamp(),
				Region:            region,
				SecurityPolicy:    tp.GetSecurityPolicy(),
				SessionAffinity:   tp.GetSessionAffinity(),
			}
			targetPools = append(targetPools, item)

			// Target Pool Instance
			for _, tpi := range tp.GetInstances() {
				item := models.TargetPoolInstance{
					TargetPoolID: tp.GetId(),
					ProjectID:    payload.ProjectID,
					InstanceName: gcputils.ResourceNameFromURL(tpi),
				}
				targetPoolInstances = append(targetPoolInstances, item)
			}
		}
	}

	// UPSERT Target Pools
	if len(targetPools) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&targetPools).
		On("CONFLICT (target_pool_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("description = EXCLUDED.description").
		Set("backup_pool = EXCLUDED.backup_pool").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("region = EXCLUDED.region").
		Set("security_policy = EXCLUDED.security_policy").
		Set("session_affinity = EXCLUDED.session_affinity").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gcp target pools",
		"project", payload.ProjectID,
		"count", count,
	)

	// UPSERT Target Pool Instances
	if len(targetPoolInstances) == 0 {
		return nil
	}

	out, err = db.DB.NewInsert().
		Model(&targetPoolInstances).
		On("CONFLICT (target_pool_id, project_id, instance_name) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gcp target pool instances",
		"project", payload.ProjectID,
		"count", count,
	)

	return nil
}
