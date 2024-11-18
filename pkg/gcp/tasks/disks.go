// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/hibiken/asynq"
	"google.golang.org/api/iterator"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/constants"
	"github.com/gardener/inventory/pkg/gcp/models"
	"github.com/gardener/inventory/pkg/gcp/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectDisks is the name of the task for collecting GCP
	// disks.
	TaskCollectDisks = "gcp:task:collect-disks"
)

// NewCollectDisksTask creates a new [asynq.Task] task for collecting GCP
// disks without specifying a payload.
func NewCollectDisksTask() *asynq.Task {
	return asynq.NewTask(TaskCollectDisks, nil)
}

// CollectDisksPayload is the payload, which is used to collect GCP disks.
type CollectDisksPayload struct {
	// ProjectID specifies the GCP project ID, which is associated with a
	// registered client.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// HandleCollectDisksTask is the handler, which collects GCP disks.
func HandleCollectDisksTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we will enqueue tasks for
	// collecting disks for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectDisks(ctx)
	}

	// Collect disks using the client associated with the project ID from
	// the payload.
	var payload CollectDisksPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectDisks(ctx, payload)
}

// enqueueCollectDisks enqueues tasks for collecting GCP disks
// for all collected GCP projects.
func enqueueCollectDisks(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	err := gcpclients.DisksClientset.Range(func(projectID string, c *gcpclients.Client[*compute.DisksClient]) error {
		p := &CollectDisksPayload{ProjectID: projectID}
		data, err := json.Marshal(p)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP disks",
				"project", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		task := asynq.NewTask(TaskCollectDisks, data)
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

// collectDisks collects the GCP disks using the client configuration
// specified in the payload.
func collectDisks(ctx context.Context, payload CollectDisksPayload) error {
	client, ok := gcpclients.DisksClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)

	logger.Info("collecting GCP disks", "project", payload.ProjectID)

	pageSize := uint32(constants.PageSize)
	partialSuccess := bool(true)
	disksRequest := computepb.AggregatedListDisksRequest{
		Project:              payload.ProjectID,
		MaxResults:           &pageSize,
		ReturnPartialSuccess: &partialSuccess,
	}
	iter := client.Client.AggregatedList(ctx, &disksRequest)

	disks := make([]models.Disk, 0)
	attachedDisks := make([]models.AttachedDisk, 0)

	for {
		pair, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			logger.Error("failed to get disks",
				"project", payload.ProjectID,
				"reason", err,
			)
			return err
		}

		items := pair.Value.Disks
		for _, i := range items {
			if i == nil {
				continue
			}
			currentDiskAttachedInstances := i.GetUsers()

			zone := utils.ResourceNameFromURL(i.GetZone())
			for _, instanceURL := range currentDiskAttachedInstances {
				attachedDisk := models.AttachedDisk{
					InstanceName: utils.ResourceNameFromURL(instanceURL),
					DiskName:     i.GetName(),
					ProjectID:    payload.ProjectID,
					Zone:         zone,
					Region:       utils.RegionFromZone(zone),
				}
				attachedDisks = append(attachedDisks, attachedDisk)
			}

			isRegional := (zone == "")

			var region string
			if isRegional {
				region = utils.ResourceNameFromURL(i.GetRegion())
			} else {
				region = utils.RegionFromZone(zone)
			}

			disk := models.Disk{
				Name:              i.GetName(),
				ProjectID:         payload.ProjectID,
				Zone:              zone,
				Region:            region,
				Description:       i.GetDescription(),
				Type:              utils.ResourceNameFromURL(i.GetType()),
				IsRegional:        isRegional,
				CreationTimestamp: i.GetCreationTimestamp(),
			}

			disks = append(disks, disk)
		}
	}

	if len(disks) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&disks).
		On("CONFLICT (name, project_id, zone) DO UPDATE").
		Set("region = EXCLUDED.region").
		Set("type = EXCLUDED.type").
		Set("description = EXCLUDED.description").
		Set("is_regional = EXCLUDED.is_regional").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert disks into db",
			"project", payload.ProjectID,
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gcp disks",
		"project", payload.ProjectID,
		"count", count,
	)

	out, err = db.DB.NewInsert().
		Model(&attachedDisks).
		On("CONFLICT (instance_name, disk_name, project_id) DO UPDATE").
		Set("zone = EXCLUDED.zone").
		Set("region = EXCLUDED.region").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert attached disks into db",
			"project", payload.ProjectID,
			"reason", err,
		)
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gcp attached disks",
		"project", payload.ProjectID,
		"count", count,
	)
	return nil
}
