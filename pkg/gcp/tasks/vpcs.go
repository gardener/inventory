// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/core/tasks"
	"github.com/gardener/inventory/pkg/gcp/constants"
	"github.com/gardener/inventory/pkg/gcp/models"
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"

	computepb "cloud.google.com/go/compute/apiv1/computepb"

	"google.golang.org/api/iterator"
)

const (
	// TaskCollectVPCs is the name of the task for collecting GCP
	// VPCs.
	TaskCollectVPCs = "gcp:task:collect-vpcs"
)

// NewCollectVPCsTask creates a new [asynq.Task] task for collecting GCP
// VPCs without specifying a payload.
func NewCollectVPCsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectVPCs, nil)
}

// CollectVPCsPayload is the payload, which is used to collect GCP VPCs.
type CollectVPCsPayload struct {
	// ProjectID specifies the GCP project ID, which is associated with a
	// registered client.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// HandleCollectVPCsTask is the handler, which collects GCP VPCs.
func HandleCollectVPCsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we will enqueue tasks for
	// collecting VPCs for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectVPCs(ctx)
	}

	// Collect VPCs using the client associated with the project ID from
	// the payload.
	var payload CollectVPCsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectVPCs(ctx, payload)
}

// enqueueCollectVPCs enqueues tasks for collecting GCP VPCs
// for all collected GCP projects.
func enqueueCollectVPCs(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	projects, err := gcputils.GetProjectsFromDB(ctx)
	if err != nil {
		logger.Error(
			"unable to retrieve projects from db",
			"reason",
			err,
		)
	}

	for _, project := range projects {
		projectID := project.ProjectID

		p := &CollectVPCsPayload{ProjectID: projectID}
		data, err := json.Marshal(p)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP VPCs",
				"project_id", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		task := asynq.NewTask(TaskCollectVPCs, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project_id", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"project_id", projectID,
		)
		return nil
	}

	return err
}

// collectVPCs collects the GCP VPCs using the client configuration
// specified in the payload.
func collectVPCs(ctx context.Context, payload CollectVPCsPayload) error {
	client, ok := gcpclients.NetworksClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(tasks.ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)

	logger.Info("collecting GCP VPCs", "project_id", payload.ProjectID)

	req := computepb.ListNetworksRequest{
		Project: payload.ProjectID,
		MaxResults: ptr.To(constants.PageSize),
	}

	vpcIter := client.Client.List(ctx, &req)

	vpcs := make([]models.VPC, 0)

	for {
		vpc, err := vpcIter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			logger.Error("unable to iterate over GCP VPCs",
				"project_id", payload.ProjectID,
				"reason", err,
			)
			continue
		}

		if err := validateVPC(vpc); err != nil {
			logger.Error("invalid vpc returned",
				"project_id", payload.ProjectID,
				"reason", err,
			)
			continue
		}

		creationTimestamp, err := time.Parse(time.RFC3339, *vpc.CreationTimestamp)
		if err != nil {
			logger.Error(
				"invalid vpc creation timestamp",
				"project_id", payload.ProjectID,
				"reason", err,
			)
			continue
		}

		item := models.VPC{
			VPCID:                *vpc.Id,
			ProjectID:            payload.ProjectID,
			Name:                 *vpc.Name,
			VPCCreationTimestamp: creationTimestamp,
			Description:          ptr.Value(vpc.Description, ""),
			GatewayIPv4:          ptr.Value(vpc.GatewayIPv4, ""),
			FirewallPolicy:       ptr.Value(vpc.FirewallPolicy, ""),
		}

		vpcs = append(vpcs, item)
	}

	if len(vpcs) == 0 {
		return nil
	}

	logger.Info("Collected vpcs", "count", len(vpcs))

		out, err := db.DB.NewInsert().
			Model(&vpcs).
			On("CONFLICT (vpc_id, project_id) DO UPDATE").
			Set("name = EXCLUDED.name").
			Set("vpc_creation_timestamp = EXCLUDED.vpc_creation_timestamp").
			Set("description = EXCLUDED.description").
			Set("gateway_ipv4 = EXCLUDED.gateway_ipv4").
			Set("firewall_policy = EXCLUDED.firewall_policy").

			Set("updated_at = EXCLUDED.updated_at").
			Returning("id").
			Exec(ctx)

		if err != nil {
			logger.Error(
				"could not insert vpcs into db",
				"project_id", payload.ProjectID,
				"reason", err,
			)
			return err
		}

		count, err := out.RowsAffected()
		if err != nil {
			return err
		}

		logger.Info(
			"populated gcp vpcs",
			"project_id", payload.ProjectID,
			"count", count,
		)

	return nil
}

func validateVPC(vpc *computepb.Network) error {
	if vpc == nil {
		return errors.New("missing vpc")
	}
	if vpc.Name == nil {
		return errors.New("missing vpc name")
	}

	if vpc.Id == nil {
		return errors.New("missing vpc id")
	}
	if vpc.CreationTimestamp == nil {
		return errors.New("missing vpc creation timestamp")
	}

	return nil
}
