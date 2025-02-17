// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/openstack/models"
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
	// Project specifies the project from which to collect.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectServersTask creates a new [asynq.Task] for collecting OpenStack
// servers, without specifying a payload.
func NewCollectServersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectServers, nil)
}

// HandleCollectServersTask handles the task for collecting OpenStack Servers.
func HandleCollectServersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Servers from all known projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectServers(ctx)
	}

	var payload CollectServersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(errors.New("no project ID specified"))
	}

	return collectServers(ctx, payload)
}

// enqueueCollectServers enqueues tasks for collecting OpenStack Servers from
// all configured OpenStack projects by creating a payload with the respective
// project ID.
func enqueueCollectServers(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.ComputeClientset.Length() == 0 {
		logger.Warn("no OpenStack compute clients found")
		return nil
	}

	err := openstackclients.ComputeClientset.Range(func(projectID string, client openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectServersPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack servers",
				"project_id", projectID,
				"reason", err,
			)
			return err
		}

		queue := asynqutils.GetQueueName(ctx)

		task := asynq.NewTask(TaskCollectServers, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project_id", projectID,
				"reason", err,
			)
			return err
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

	if err != nil {
		logger.Error(
			"couldn't enqueue collection of servers",
			"reason", err,
		)
	}

	return nil
}

// collectServer collects the OpenStack servers from the specified project,
// using the client associated with the project ID in the given payload.
func collectServers(ctx context.Context, payload CollectServersPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.ComputeClientset.Get(payload.ProjectID)
	if !ok {
		logger.Error(
			"no client for given project",
			"project_id", payload.ProjectID,
		)
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	region := client.Region
	domain := client.Domain
	projectID := payload.ProjectID

	logger.Info(
		"collecting OpenStack servers",
		"project_id", client.ProjectID,
		"domain", client.Domain,
		"region", client.Region,
		"named_credentials", client.NamedCredentials,
	)

	items := make([]models.Server, 0)

	err := servers.List(client.Client, nil).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
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
						Domain:           domain,
						Region:           region,
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
			"project_id", projectID,
			"region", region,
			"domain", domain,
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack servers",
		"project_id", projectID,
		"region", region,
		"domain", domain,
		"count", count,
	)

	return nil
}
