// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/openstack/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectSubnets is the name of the task for collecting OpenStack
	// Subnets.
	TaskCollectSubnets = "openstack:task:collect-subnets"
)

// CollectSubnetsPayload represents the payload, which specifies
// where to collect OpenStack Subnets from.
type CollectSubnetsPayload struct {
	// Project specifies the project from which to collect.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectSubnetsTask creates a new [asynq.Task] for collecting OpenStack
// Subnets, without specifying a payload.
func NewCollectSubnetsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectSubnets, nil)
}

// HandleCollectSubnetsTask handles the task for collecting OpenStack Subnets.
func HandleCollectSubnetsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Subnets from all known projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectSubnets(ctx)
	}

	var payload CollectSubnetsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectSubnets(ctx, payload)

}

// enqueueCollectSubnets enqueues tasks for collecting OpenStack Subnets from
// all configured OpenStack network clients by creating a payload with the respective
// project ID.
func enqueueCollectSubnets(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.NetworkClientset.Length() == 0 {
		logger.Warn("no OpenStack subnet clients found")
		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.NetworkClientset.Range(func(projectID string, client openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectSubnetsPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack subnets",
				"project_id", projectID,
				"reason", err,
			)
			return err
		}

		task := asynq.NewTask(TaskCollectSubnets, data)
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

// collectSubnets collects the OpenStack Subnets from the specified project id,
// using the client associated with the project ID in the given payload.
func collectSubnets(ctx context.Context, payload CollectSubnetsPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.NetworkClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	region := client.Region
	domain := client.Domain
	projectID := payload.ProjectID

	logger.Info(
		"collecting OpenStack subnets",
		"project_id", client.ProjectID,
		"domain", client.Domain,
		"region", client.Region,
		"named_credentials", client.NamedCredentials,
	)

	items := make([]models.Subnet, 0)

	err := subnets.List(client.Client, nil).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
				subnetList, err := subnets.ExtractSubnets(page)

				if err != nil {
					logger.Error(
						"could not extract subnet pages",
						"reason", err,
					)
					return false, err
				}

				for _, s := range subnetList {
					item := models.Subnet{
						SubnetID:     s.ID,
						Name:         s.Name,
						ProjectID:    s.TenantID,
						Domain:       domain,
						Region:       region,
						NetworkID:    s.NetworkID,
						GatewayIP:    s.GatewayIP,
						CIDR:         s.CIDR,
						SubnetPoolID: s.SubnetPoolID,
						EnableDHCP:   s.EnableDHCP,
						IPVersion:    s.IPVersion,
						Description:  s.Description,
					}

					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract subnet pages",
			"reason", err,
		)
		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (subnet_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("domain = EXCLUDED.domain").
		Set("region = EXCLUDED.region").
		Set("network_id = EXCLUDED.network_id").
		Set("gateway_ip = EXCLUDED.gateway_ip").
		Set("cidr = EXCLUDED.cidr").
		Set("subnet_pool_id = EXCLUDED.subnet_pool_id").
		Set("enable_dhcp = EXCLUDED.enable_dhcp").
		Set("ip_version = EXCLUDED.ip_version").
		Set("description = EXCLUDED.description").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert Subnets into db",
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
		"populated openstack subnets",
		"project_id", projectID,
		"region", region,
		"domain", domain,
		"count", count,
	)

	return nil
}
