// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"

	computepb "cloud.google.com/go/compute/apiv1/computepb"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"github.com/hibiken/asynq"
	"google.golang.org/api/iterator"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/constants"
	"github.com/gardener/inventory/pkg/gcp/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
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

	err := gcpclients.ProjectsClientset.Range(func(projectID string, _ *gcpclients.Client[*resourcemanager.ProjectsClient]) error {
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
	})

	return err
}

// collectVPCs collects the GCP VPCs using the client configuration
// specified in the payload.
func collectVPCs(ctx context.Context, payload CollectVPCsPayload) error {
	client, ok := gcpclients.NetworksClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)

	logger.Info("collecting GCP VPCs", "project_id", payload.ProjectID)

	pageSize := uint32(constants.PageSize)
	req := computepb.ListNetworksRequest{
		Project:    payload.ProjectID,
		MaxResults: &pageSize,
	}

	vpcIter := client.Client.List(ctx, &req)

	vpcs := make([]models.VPC, 0)

	for {
		vpc, err := vpcIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			logger.Error("failed to get GCP VPCs",
				"project_id", payload.ProjectID,
				"reason", err,
			)
			return err
		}

		logger.Info(
			"mtu for vpc",
			"vpc", vpc.GetName(),
			"mtu", vpc.GetMtu(),
		)
		item := models.VPC{
			VPCID:                    vpc.GetId(),
			ProjectID:                payload.ProjectID,
			Name:                     vpc.GetName(),
			VPCCreationTimestamp:     vpc.GetCreationTimestamp(),
			Description:              vpc.GetDescription(),
			GatewayIPv4:              vpc.GetGatewayIPv4(),
			FirewallPolicy:           vpc.GetFirewallPolicy(),
			MaxTransmissionUnitBytes: vpc.GetMtu(),
		}
		vpcs = append(vpcs, item)
	}

	if len(vpcs) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&vpcs).
		On("CONFLICT (vpc_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("vpc_creation_timestamp = EXCLUDED.vpc_creation_timestamp").
		Set("description = EXCLUDED.description").
		Set("gateway_ipv4 = EXCLUDED.gateway_ipv4").
		Set("firewall_policy = EXCLUDED.firewall_policy").
		Set("max_transmission_units_bytes = EXCLUDED.max_transmission_units_bytes").
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
