// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

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

const (
	// TaskCollectSubnets is the name of the task for collecting GCP
	// subnets.
	TaskCollectSubnets = "gcp:task:collect-subnets"
)

// NewCollectSubnetsTask creates a new [asynq.Task] task for collecting GCP
// subnets without specifying a payload.
func NewCollectSubnetsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectSubnets, nil)
}

// CollectSubnetsPayload is the payload, which is used to collect GCP subnets.
type CollectSubnetsPayload struct {
	// ProjectID specifies the GCP project ID, which is associated with a
	// registered client.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// HandleCollectSubnetsTask is the handler, which collects GCP subnets.
func HandleCollectSubnetsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we will enqueue tasks for
	// collecting subnets for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectSubnets(ctx)
	}

	// Collect subnets using the client associated with the project ID from
	// the payload.
	var payload CollectSubnetsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectSubnets(ctx, payload)
}

// enqueueCollectSubnets enqueues tasks for collecting GCP subnets
// for all collected GCP projects.
func enqueueCollectSubnets(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	err := gcpclients.SubnetworksClientset.Range(func(projectID string, c *gcpclients.Client[*compute.SubnetworksClient]) error {
		p := &CollectSubnetsPayload{ProjectID: projectID}
		data, err := json.Marshal(p)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP subnets",
				"project", projectID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		task := asynq.NewTask(TaskCollectSubnets, data)
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

// collectSubnets collects the GCP subnets using the client configuration
// specified in the payload.
func collectSubnets(ctx context.Context, payload CollectSubnetsPayload) error {
	client, ok := gcpclients.SubnetworksClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	logger := asynqutils.GetLogger(ctx)

	logger.Info("collecting GCP subnets", "project", payload.ProjectID)

	pageSize := uint32(constants.PageSize)
	partialSuccess := true
	req := computepb.AggregatedListSubnetworksRequest{
		Project:              gcputils.ProjectFQN(payload.ProjectID),
		MaxResults:           &pageSize,
		ReturnPartialSuccess: &partialSuccess,
	}

	subnetIter := client.Client.AggregatedList(ctx, &req)
	if subnetIter == nil {
		err := fmt.Errorf("%w", "no iterator returned")
		logger.Error(
			"unable to list subnets",
			"project", payload.ProjectID,
			"reason", err,
		)
		return err
	}

	items := make([]models.Subnet, 0)

	for {
		pair, err := subnetIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			logger.Error("failed to get subnets",
				"project", payload.ProjectID,
				"reason", err,
			)
			return err
		}

		// we do not need the key, as it is the region and we get that in the values as well
		subnets := pair.Value.GetSubnetworks()
		for _, i := range subnets {
            CIDRRange := i.GetIpCidrRange()
			_, _, err := net.ParseCIDR(CIDRRange)
			if err != nil {
				logger.Warn(
                    "invalid IP CIDR found", 
                    "cidr", CIDRRange,
                    "reason", err,
                )
                return err
			}

			item := models.Subnet{
				SubnetID:          i.GetId(),
				VPCName:           gcputils.ResourceNameFromURL(i.GetNetwork()),
				ProjectID:         payload.ProjectID,
				Name:              i.GetName(),
				Region:            i.GetRegion(),
				CreationTimestamp: i.GetCreationTimestamp(),
				Description:       i.GetDescription(),
				IPv4CIDRRange:     CIDRRange,
				Gateway:           i.GetGatewayAddress(),
				Purpose:           i.GetPurpose(),
			}

			items = append(items, item)
		}
	}

	logger.Info("subnets",
		"count", len(items))

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (subnet_id, vpc_name, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("region = EXCLUDED.region").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("description = EXCLUDED.description").
		Set("ipv4_cidr_range = EXCLUDED.ipv4_cidr_range").
		Set("gateway = EXCLUDED.gateway").
		Set("purpose = EXCLUDED.purpose").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert subnets into db",
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
		"populated gcp subnets",
		"project", payload.ProjectID,
		"count", count,
	)

	return nil
}
