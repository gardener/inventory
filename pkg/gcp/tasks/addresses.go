// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"net"

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
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// TaskCollectAddresses is the name of the task for collecting global and
// regional static IP addresses.
const TaskCollectAddresses = "gcp:task:collect-addresses"

// CollectAddressesPayload is the payload used for collecting global and
// regional static IP addresses.
type CollectAddressesPayload struct {
	// ProjectID specifies the globally unique project id from which to
	// collect.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectAddressesTask creates a new [asynq.Task] for collecting global and
// regional static IP addresses, without specifying a payload.
func NewCollectAddressesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectAddresses, nil)
}

// HandleCollectAddressesTask is the handler, which collects static global and
// regional IP addresses.
func HandleCollectAddressesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting from all registered projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectAddresses(ctx)
	}

	var payload CollectAddressesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectAddresses(ctx, payload)
}

// enqueueCollectAddresses enqueues tasks for collecting global and regional static IP
// addresses for all known projects.
func enqueueCollectAddresses(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if gcpclients.AddressesClientset.Length() == 0 && gcpclients.GlobalAddressesClientset.Length() == 0 {
		logger.Warn("no GCP addresses clients found")

		return nil
	}

	// Enqueue tasks for all registered GCP Projects. Same projects are
	// registered for the regional and global addresses clients, so here we
	// can iterate through just one of the registries.
	queue := asynqutils.GetQueueName(ctx)
	err := gcpclients.AddressesClientset.Range(func(projectID string, _ *gcpclients.Client[*compute.AddressesClient]) error {
		payload := CollectAddressesPayload{ProjectID: projectID}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP addresses",
				"project", projectID,
				"reason", err,
			)

			return registry.ErrContinue
		}
		task := asynq.NewTask(TaskCollectAddresses, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
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

// getRegionalAddresses fetches the regional static IP addresses for the
// project specified in the payload.
func getRegionalAddresses(ctx context.Context, payload CollectAddressesPayload) ([]*computepb.Address, error) {
	client, ok := gcpclients.AddressesClientset.Get(payload.ProjectID)
	if !ok {
		return nil, asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	partialSuccess := true
	pageSize := uint32(constants.PageSize)
	req := &computepb.AggregatedListAddressesRequest{
		Project:              gcputils.ProjectFQN(payload.ProjectID),
		ReturnPartialSuccess: &partialSuccess,
		MaxResults:           &pageSize,
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting gcp regional addresses", "project_id", payload.ProjectID)

	it := client.Client.AggregatedList(ctx, req)
	items := make([]*computepb.Address, 0)
	for {
		pair, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		items = append(items, pair.Value.Addresses...)
	}

	return items, nil
}

// getGlobalAddresses fetch the global ANYCAST addresses for the project
// specified in the payload.
func getGlobalAddresses(ctx context.Context, payload CollectAddressesPayload) ([]*computepb.Address, error) {
	client, ok := gcpclients.GlobalAddressesClientset.Get(payload.ProjectID)
	if !ok {
		return nil, asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	partialSuccess := true
	pageSize := uint32(constants.PageSize)
	req := &computepb.ListGlobalAddressesRequest{
		Project:              payload.ProjectID,
		ReturnPartialSuccess: &partialSuccess,
		MaxResults:           &pageSize,
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting gcp global addresses", "project_id", payload.ProjectID)

	it := client.Client.List(ctx, req)
	items := make([]*computepb.Address, 0)
	for {
		item, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

// collectAddresses collects the global and regional static IP addresses for the
// project specified in the payload.
func collectAddresses(ctx context.Context, payload CollectAddressesPayload) error {
	logger := asynqutils.GetLogger(ctx)

	regional, err := getRegionalAddresses(ctx, payload)
	if err != nil {
		return err
	}

	global, err := getGlobalAddresses(ctx, payload)
	if err != nil {
		return err
	}

	addresses := []struct {
		items    []*computepb.Address
		isGlobal bool
	}{
		{
			items:    regional,
			isGlobal: false,
		},
		{
			items:    global,
			isGlobal: true,
		},
	}

	items := make([]models.Address, 0)
	for _, pair := range addresses {
		for _, addr := range pair.items {
			ip := net.ParseIP(addr.GetAddress())
			item := models.Address{
				Address:           ip,
				AddressType:       addr.GetAddressType(),
				IsGlobal:          pair.isGlobal,
				CreationTimestamp: addr.GetCreationTimestamp(),
				Description:       addr.GetDescription(),
				AddressID:         addr.GetId(),
				ProjectID:         payload.ProjectID,
				Region:            gcputils.ResourceNameFromURL(addr.GetRegion()),
				IPVersion:         addr.GetIpVersion(),
				IPv6EndpointType:  addr.GetIpv6EndpointType(),
				Name:              addr.GetName(),
				Network:           gcputils.ResourceNameFromURL(addr.GetNetwork()),
				NetworkTier:       addr.GetNetworkTier(),
				Subnetwork:        gcputils.ResourceNameFromURL(addr.GetSubnetwork()),
				PrefixLength:      int(addr.GetPrefixLength()),
				Purpose:           addr.GetPurpose(),
				SelfLink:          addr.GetSelfLink(),
				Status:            addr.GetStatus(),
			}
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (project_id, address_id) DO UPDATE").
		Set("address = EXCLUDED.address").
		Set("address_type = EXCLUDED.address_type").
		Set("is_global = EXCLUDED.is_global").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("description = EXCLUDED.description").
		Set("region = EXCLUDED.region").
		Set("ip_version = EXCLUDED.ip_version").
		Set("ipv6_endpoint_type = EXCLUDED.ipv6_endpoint_type").
		Set("name = EXCLUDED.name").
		Set("network = EXCLUDED.network").
		Set("network_tier = EXCLUDED.network_tier").
		Set("subnetwork = EXCLUDED.subnetwork").
		Set("prefix_length = EXCLUDED.prefix_length").
		Set("purpose = EXCLUDED.purpose").
		Set("status = EXCLUDED.status").
		Set("self_link = EXCLUDED.self_link").
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
		"populated gcp addresses",
		"project", payload.ProjectID,
		"count", count,
	)

	return nil
}
