// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
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
	// TaskCollectServers is the name of the task for collecting OpenStack
	// servers.
	TaskCollectServers = "openstack:task:collect-servers"
)

// CollectServersPayload represents the payload, which specifies
// where to collect OpenStack Servers from.
type CollectServersPayload struct {
	// Scope specifies the client scope for which to collect.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectServersTask creates a new [asynq.Task] for collecting OpenStack
// servers, without specifying a payload.
func NewCollectServersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectServers, nil)
}

// HandleCollectServersTask handles the task for collecting OpenStack Servers.
func HandleCollectServersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Servers from all configured server clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectServers(ctx)
	}

	var payload CollectServersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectServers(ctx, payload)
}

// enqueueCollectServers enqueues tasks for collecting OpenStack Servers from
// all configured OpenStack server clients by creating a payload with the respective
// client scope.
func enqueueCollectServers(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.ComputeClientset.Length() == 0 {
		logger.Warn("no OpenStack compute clients found")

		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.ComputeClientset.Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectServersPayload{
			Scope: scope,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack servers",
				"project", scope.Project,
				"domain", scope.Domain,
				"region", scope.Region,
				"reason", err,
			)

			return err
		}

		task := asynq.NewTask(TaskCollectServers, data)
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

// collectServer collects the OpenStack servers,
// using the client associated with the client scope in the given payload.
func collectServers(ctx context.Context, payload CollectServersPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.ComputeClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack servers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			serversDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		key := metrics.Key(
			TaskCollectServers,
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	items := make([]models.Server, 0)

	opts := servers.ListOpts{
		TenantID: client.ProjectID,
	}
	err := servers.List(client.Client, opts).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				serverList, err := servers.ExtractServers(page)

				if err != nil {
					logger.Error(
						"could not extract server pages",
						"reason", err,
					)

					return false, err
				}

				for _, s := range serverList {
					item := models.Server{
						ServerID:         s.ID,
						Name:             s.Name,
						ProjectID:        s.TenantID,
						Domain:           client.Domain,
						Region:           client.Region,
						UserID:           s.UserID,
						AvailabilityZone: s.AvailabilityZone,
						Status:           s.Status,
						TimeCreated:      s.Created,
						TimeUpdated:      s.Updated,
					}

					imageID, ok := s.Image["id"]
					if ok {
						image, ok := imageID.(string)
						if ok {
							item.ImageID = image
						}
					}

					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract server pages",
			"reason", err,
		)

		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (server_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("domain = EXCLUDED.domain").
		Set("region = EXCLUDED.region").
		Set("user_id = EXCLUDED.user_id").
		Set("availability_zone = EXCLUDED.availability_zone").
		Set("status = EXCLUDED.status").
		Set("image_id = EXCLUDED.image_id").
		Set("server_created_at = EXCLUDED.server_created_at").
		Set("server_updated_at = EXCLUDED.server_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert servers into db",
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
		"populated openstack servers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}
