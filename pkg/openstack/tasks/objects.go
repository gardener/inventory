// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/openstack/models"
	openstackutils "github.com/gardener/inventory/pkg/openstack/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectObjects is the name of the task for collecting OpenStack
	// Objects.
	TaskCollectObjects = "openstack:task:collect-objects"
)

// CollectObjectsPayload represents the payload, which specifies
// where to collect OpenStack Objects from.
type CollectObjectsPayload struct {
	// Scope specifies the client scope for which to collect.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectObjectsTask creates a new [asynq.Task] for collecting OpenStack
// Objects, without specifying a payload.
func NewCollectObjectsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectObjects, nil)
}

// HandleCollectObjectsTask handles the task for collecting OpenStack Objects.
func HandleCollectObjectsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Objects from all configured object clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectObjects(ctx)
	}

	var payload CollectObjectsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectObjects(ctx, payload)
}

// enqueueCollectObjects enqueues tasks for collecting OpenStack Objects from
// all configured OpenStack Object clients by creating a payload with the respective
// client scope.
func enqueueCollectObjects(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.ObjectStorageClientset.Length() == 0 {
		logger.Warn("no OpenStack object storage clients found")

		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.ObjectStorageClientset.
		Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
			payload := CollectObjectsPayload{
				Scope: scope,
			}
			data, err := json.Marshal(payload)
			if err != nil {
				logger.Error(
					"failed to marshal payload for OpenStack objects",
					"project", scope.Project,
					"domain", scope.Domain,
					"region", scope.Region,
					"reason", err,
				)

				return err
			}

			task := asynq.NewTask(TaskCollectObjects, data)
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

// collectObject collects the OpenStack Objects,
// using the client associated with the client scope in the given payload.
func collectObjects(ctx context.Context, payload CollectObjectsPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.ObjectStorageClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack objects",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	containerNames := make([]string, 0)
	items := make([]models.Object, 0)

	err := containers.List(client.Client, nil).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				containerNameList, err := containers.ExtractNames(page)

				if err != nil {
					logger.Error(
						"could not extract container pages",
						"reason", err,
					)

					return false, err
				}
				containerNames = append(containerNames, containerNameList...)

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract container pages",
			"reason", err,
		)

		return err
	}

	for _, name := range containerNames {
		err = objects.List(client.Client, name, nil).
			EachPage(ctx,
				func(_ context.Context, page pagination.Page) (bool, error) {
					objectList, err := objects.ExtractInfo(page)

					if err != nil {
						logger.Error(
							"could not extract object pages",
							"reason", err,
						)

						return false, err
					}

					for _, o := range objectList {
						item := models.Object{
							Name:          o.Name,
							ContainerName: name,
							ProjectID:     client.Project,
							ContentType:   o.ContentType,
							LastModified:  o.LastModified,
							IsLatest:      o.IsLatest,
						}

						items = append(items, item)
					}

					return true, nil
				})

		if err != nil {
			logger.Warn(
				"could not extract object pages",
				"container",
				name,
				"reason", err,
			)

			continue
		}
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, container_name, project_id) DO UPDATE").
		Set("content_type = EXCLUDED.content_type").
		Set("last_modified = EXCLUDED.last_modified").
		Set("is_latest = EXCLUDED.is_latest").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert objects into db",
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack objects",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}
