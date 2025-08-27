// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
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
	dbutils "github.com/gardener/inventory/pkg/utils/db"
)

const (
	// TaskCollectContainers is the name of the task for collecting OpenStack
	// Containers.
	TaskCollectContainers = "openstack:task:collect-containers"
)

// CollectContainersPayload represents the payload, which specifies
// where to collect OpenStack Containers from.
type CollectContainersPayload struct {
	// Scope specifies the client scope for which to collect.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectContainersTask creates a new [asynq.Task] for collecting OpenStack
// Containers, without specifying a payload.
func NewCollectContainersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectContainers, nil)
}

// HandleCollectContainersTask handles the task for collecting OpenStack Containers.
func HandleCollectContainersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Containers from all configured object clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectContainers(ctx)
	}

	var payload CollectContainersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectContainers(ctx, payload)
}

// enqueueCollectContainers enqueues tasks for collecting OpenStack Containers from
// all configured OpenStack Container clients by creating a payload with the respective
// client scope.
func enqueueCollectContainers(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.ObjectStorageClientset.Length() == 0 {
		logger.Warn("no OpenStack object storage clients found")

		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.ObjectStorageClientset.
		Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
			payload := CollectContainersPayload{
				Scope: scope,
			}
			data, err := json.Marshal(payload)
			if err != nil {
				logger.Error(
					"failed to marshal payload for OpenStack containers",
					"project", scope.Project,
					"domain", scope.Domain,
					"region", scope.Region,
					"reason", err,
				)

				return err
			}

			task := asynq.NewTask(TaskCollectContainers, data)
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

// collectContainer collects the OpenStack Containers,
// using the client associated with the client scope in the given payload.
func collectContainers(ctx context.Context, payload CollectContainersPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.ObjectStorageClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack containers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			containersDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		key := metrics.Key(
			TaskCollectContainers,
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	items := make([]models.Container, 0)

	projects, err := dbutils.GetResourcesFromDB[models.Project](ctx)

	if err != nil {
		logger.Error(
			"could not get projects from db",
			"reason", err,
		)

		return err
	}

	err = containers.List(client.Client, nil).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				extractedContainers, err := containers.ExtractInfo(page)
				if err != nil {
					logger.Error(
						"could not extract container pages",
						"reason", err,
					)

					return false, err
				}

				for _, container := range extractedContainers {
					project, err := openstackutils.MatchScopeToProject(client.ClientScope, projects)
					if err != nil {
						logger.Error(
							"could not get project for container",
							"reason", err,
						)
					}

					item := models.Container{
						Name:        container.Name,
						ProjectID:   project.ProjectID,
						Bytes:       container.Bytes,
						ObjectCount: container.Count,
					}

					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract container pages",
			"reason", err,
		)

		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, project_id) DO UPDATE").
		Set("bytes = EXCLUDED.bytes").
		Set("object_count = EXCLUDED.object_count").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert containers into db",
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
		"populated openstack containers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}
