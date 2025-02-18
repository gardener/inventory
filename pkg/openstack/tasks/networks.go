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

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/openstack/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectNetworks is the name of the task for collecting OpenStack
	// Networks.
	TaskCollectNetworks = "openstack:task:collect-networks"
)

// CollectNetworksPayload represents the payload, which specifies
// where to collect OpenStack Networks from.
type CollectNetworksPayload struct {
	// Project specifies the project from which to collect.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectNetworksTask creates a new [asynq.Task] for collecting OpenStack
// Networks, without specifying a payload.
func NewCollectNetworksTask() *asynq.Task {
	return asynq.NewTask(TaskCollectNetworks, nil)
}

// HandleCollectNetworksTask handles the task for collecting OpenStack Networks.
func HandleCollectNetworksTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Networks from all known projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectNetworks(ctx)
	}

	var payload CollectNetworksPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectNetworks(ctx, payload)

}

// enqueueCollectNetworks enqueues tasks for collecting OpenStack Networks from
// all configured OpenStack projects by creating a payload with the respective
// project ID.
func enqueueCollectNetworks(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.NetworkClientset.Length() == 0 {
		logger.Warn("no OpenStack network clients found")
		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.NetworkClientset.Range(func(projectID string, client openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectNetworksPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack networks",
				"project_id", projectID,
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
}

// collectNetworks collects the OpenStack Networks from the specified project id,
// using the client associated with the project ID in the given payload.
func collectNetworks(ctx context.Context, payload CollectNetworksPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.NetworkClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	region := client.Region
	domain := client.Domain
	projectID := payload.ProjectID

	logger.Info(
		"collecting OpenStack networks",
		"project_id", client.ProjectID,
		"domain", client.Domain,
		"region", client.Region,
		"named_credentials", client.NamedCredentials,
	)

	items := make([]models.Network, 0)

	err := networks.List(client.Client, nil).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
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
						Domain:      domain,
						Region:      region,
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
		"populated openstack networks",
		"project_id", projectID,
		"region", region,
		"domain", domain,
		"count", count,
	)

	return nil
}
