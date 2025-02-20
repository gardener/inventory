// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hibiken/asynq"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/openstack/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectLoadBalancers is the name of the task for collecting OpenStack
	// LoadBalancers.
	TaskCollectLoadBalancers = "openstack:task:collect-loadbalancers"
)

// CollectLoadBalancersPayload represents the payload, which specifies
// where to collect OpenStack LoadBalancers from.
type CollectLoadBalancersPayload struct {
	// Project specifies the project from which to collect.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// NewCollectLoadBalancersTask creates a new [asynq.Task] for collecting OpenStack
// LoadBalancers, without specifying a payload.
func NewCollectLoadBalancersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectLoadBalancers, nil)
}

// HandleCollectLoadBalancersTask handles the task for collecting OpenStack LoadBalancers.
func HandleCollectLoadBalancersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack LoadBalancers from all known projects.
	data := t.Payload()
	if data == nil {
		return enqueueCollectLoadBalancers(ctx)
	}

	var payload CollectLoadBalancersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectLoadBalancers(ctx, payload)
}

// enqueueCollectLoadBalancers enqueues tasks for collecting OpenStack Loadbalancers from
// all configured OpenStack projects by creating a payload with the respective
// project ID.
func enqueueCollectLoadBalancers(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.LoadBalancerClientset.Length() == 0 {
		logger.Warn("no OpenStack loadbalancer clients found")
		return nil
	}

	return openstackclients.LoadBalancerClientset.Range(func(projectID string, client openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectLoadBalancersPayload{
			ProjectID: projectID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack load balancers",
				"project_id", projectID,
				"reason", err,
			)
			return err
		}

		task := asynq.NewTask(TaskCollectLoadBalancers, data)
		info, err := asynqclient.Client.Enqueue(task)
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

// collectLoadBalancers collects the OpenStack LoadBalancers from the specified project id,
// using the client associated with the project ID in the given payload.
func collectLoadBalancers(ctx context.Context, payload CollectLoadBalancersPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.LoadBalancerClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	region := client.Region
	domain := client.Domain
	projectID := payload.ProjectID

	logger.Info(
		"collecting OpenStack load balancers",
		"project_id", client.ProjectID,
		"domain", client.Domain,
		"region", client.Region,
		"named_credentials", client.NamedCredentials,
	)

	items := make([]models.LoadBalancer, 0)

	err := loadbalancers.List(client.Client, nil).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
				lbList, err := loadbalancers.ExtractLoadBalancers(page)

				if err != nil {
					logger.Error(
						"could not extract load balancers pages",
						"reason", err,
					)
					return false, err
				}

				for _, lb := range lbList {
					item := models.LoadBalancer{
						LoadBalancerID: lb.ID,
						Name:           lb.Name,
						ProjectID:      lb.ProjectID,
						Domain:         domain,
						Region:         region,
						Status:         lb.OperatingStatus,
						Description:    lb.Description,
						Provider:       lb.Provider,
						VipAddress:     lb.VipAddress,
						VipNetworkID:   lb.VipNetworkID,
						VipSubnetID:    lb.VipSubnetID,
						TimeCreated:    lb.CreatedAt,
						TimeUpdated:    lb.UpdatedAt,
					}

					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract load balancer pages",
			"reason", err,
		)
		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (loadbalancer_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("domain = EXCLUDED.domain").
		Set("region = EXCLUDED.region").
		Set("status = EXCLUDED.status").
		Set("provider = EXCLUDED.provider").
		Set("vip_address = EXCLUDED.vip_address").
		Set("vip_network_id = EXCLUDED.vip_network_id").
		Set("vip_subnet_id = EXCLUDED.vip_subnet_id").
		Set("description = EXCLUDED.description").
		Set("loadbalancer_created_at = EXCLUDED.loadbalancer_created_at").
		Set("loadbalancer_updated_at = EXCLUDED.loadbalancer_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert load balancers into db",
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
		"populated openstack load balancers",
		"project_id", projectID,
		"region", region,
		"domain", domain,
		"count", count,
	)

	return nil
}
