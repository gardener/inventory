// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/metrics"
	"github.com/gardener/inventory/pkg/openstack/models"
	openstackutils "github.com/gardener/inventory/pkg/openstack/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectNetworks is the name of the task for collecting OpenStack
	// Networks.
	TaskCollectNetworks = "openstack:task:collect-networks"
)

// CollectNetworksPayload represents the payload, which specifies
// which client to collect OpenStack Networks with.
type CollectNetworksPayload struct {
	// Scope specifies the client scope for which to collect.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectNetworksTask creates a new [asynq.Task] for collecting OpenStack
// Networks, without specifying a payload.
func NewCollectNetworksTask() *asynq.Task {
	return asynq.NewTask(TaskCollectNetworks, nil)
}

// HandleCollectNetworksTask handles the task for collecting OpenStack Networks.
func HandleCollectNetworksTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Networks for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectNetworks(ctx)
	}

	var payload CollectNetworksPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectNetworks(ctx, payload)
}

// enqueueCollectNetworks enqueues tasks for collecting OpenStack Networks from
// all configured OpenStack network clients by creating a payload with the respective
// client scope.
func enqueueCollectNetworks(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.NetworkClientset.Length() == 0 {
		logger.Warn("no OpenStack network clients found")

		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.NetworkClientset.Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectNetworksPayload{
			Scope: scope,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack networks",
				"project", scope.Project,
				"domain", scope.Domain,
				"region", scope.Region,
				"reason", err,
			)

			return err
		}

		task := asynq.NewTask(TaskCollectNetworks, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project", scope.Project,
				"domain", scope.Domain,
				"region", scope.Region,
				"reason", err,
			)

			return err
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"project", scope.Project,
			"domain", scope.Domain,
			"region", scope.Region,
		)

		return nil
	})
}

// collectNetworks collects the OpenStack Networks,
// using the client associated with the scope in the given payload.
func collectNetworks(ctx context.Context, payload CollectNetworksPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.NetworkClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack networks",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			networksDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		key := metrics.Key(
			TaskCollectNetworks,
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	items := make([]models.Network, 0)

	opts := networks.ListOpts {
		ProjectID: client.ClientScope.ProjectID,
	}
	err := networks.List(client.Client, opts).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				networkList, err := networks.ExtractNetworks(page)

				if err != nil {
					logger.Error(
						"could not extract networks pages",
						"reason", err,
					)

					return false, err
				}

				for _, n := range networkList {
					item := models.Network{
						NetworkID:   n.ID,
						Name:        n.Name,
						ProjectID:   n.TenantID,
						Domain:      client.Domain,
						Region:      client.Region,
						Status:      n.Status,
						Shared:      n.Shared,
						Description: n.Description,
						TimeCreated: n.CreatedAt,
						TimeUpdated: n.UpdatedAt,
					}

					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract network pages",
			"reason", err,
		)

		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (network_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("domain = EXCLUDED.domain").
		Set("region = EXCLUDED.region").
		Set("status = EXCLUDED.status").
		Set("shared = EXCLUDED.shared").
		Set("description = EXCLUDED.description").
		Set("network_created_at = EXCLUDED.network_created_at").
		Set("network_updated_at = EXCLUDED.network_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert networks into db",
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack networks",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}
