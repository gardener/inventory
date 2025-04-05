// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
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
	// TaskCollectProjects is the name of the task for collecting OpenStack
	// Projects.
	TaskCollectProjects = "openstack:task:collect-projects"
)

// CollectProjectsPayload represents the payload, which specifies
// where to collect OpenStack Projects from.
type CollectProjectsPayload struct {
	// Scope specifies the scope of the client to be used.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectProjectsTask creates a new [asynq.Task] for collecting OpenStack
// Projects, without specifying a payload.
func NewCollectProjectsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectProjects, nil)
}

// HandleCollectProjectsTask handles the task for collecting OpenStack Projects.
func HandleCollectProjectsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Projects for all configured identity clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectProjects(ctx)
	}

	var payload CollectProjectsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectProjects(ctx, payload)
}

// enqueueCollectProjects enqueues tasks for collecting OpenStack Projects from
// all configured OpenStack Projects by creating a payload with the respective
// Project ID.
func enqueueCollectProjects(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.IdentityClientset.Length() == 0 {
		logger.Warn("no OpenStack identity clients found")
		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.IdentityClientset.Range(func(scope openstackclients.ClientScope, client openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectProjectsPayload{
			Scope: scope,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack projects",
				"scope", scope,
				"reason", err,
			)
			return err
		}

		task := asynq.NewTask(TaskCollectProjects, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"scope", scope,
				"reason", err,
			)
			return err
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"scope", scope,
		)

		return nil
	})
}

// collectProjects collects the OpenStack Projects from the specified project,
// using the identity client associated with the project ID in the given payload.
func collectProjects(ctx context.Context, payload CollectProjectsPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.IdentityClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack projects",
		"scope", payload.Scope,
	)

	items := make([]models.Project, 0)

	err := projects.ListAvailable(client.Client).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
				projectList, err := projects.ExtractProjects(page)

				if err != nil {
					logger.Error(
						"could not extract project pages",
						"reason", err,
					)
					return false, err
				}

				for _, p := range projectList {
					item := models.Project{
						ProjectID:   p.ID,
						Name:        p.Name,
						Domain:      client.Domain,
						Region:      client.Region,
						ParentID:    p.ParentID,
						Description: p.Description,
						Enabled:     p.Enabled,
						IsDomain:    p.IsDomain,
					}

					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract project pages",
			"reason", err,
		)
		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("domain = EXCLUDED.domain").
		Set("region = EXCLUDED.region").
		Set("parent_id = EXCLUDED.parent_id").
		Set("description = EXCLUDED.description").
		Set("enabled = EXCLUDED.enabled").
		Set("is_domain = EXCLUDED.is_domain").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert projects into db",
			"scope", payload.Scope,
			"reason", err,
		)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack projects",
		"scope", payload.Scope,
		"count", count,
	)

	return nil
}
