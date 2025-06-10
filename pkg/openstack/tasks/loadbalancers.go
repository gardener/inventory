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
	openstackutils "github.com/gardener/inventory/pkg/openstack/utils"
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
	// Scope specifies the project scope to use for collection.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectLoadBalancersTask creates a new [asynq.Task] for collecting OpenStack
// LoadBalancers, without specifying a payload.
func NewCollectLoadBalancersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectLoadBalancers, nil)
}

// HandleCollectLoadBalancersTask handles the task for collecting OpenStack LoadBalancers.
func HandleCollectLoadBalancersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack LoadBalancers for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectLoadBalancers(ctx)
	}

	var payload CollectLoadBalancersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectLoadBalancers(ctx, payload)
}

// enqueueCollectLoadBalancers enqueues tasks for collecting OpenStack Loadbalancers from
// all configured OpenStack clients by creating a payload with the respective
// client scope.
func enqueueCollectLoadBalancers(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.LoadBalancerClientset.Length() == 0 {
		logger.Warn("no OpenStack loadbalancer clients found")

		return nil
	}

	return openstackclients.LoadBalancerClientset.
		Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
			payload := CollectLoadBalancersPayload{
				Scope: scope,
			}
			data, err := json.Marshal(payload)
			if err != nil {
				logger.Error(
					"failed to marshal payload for OpenStack load balancers",
					"project", scope.Project,
					"domain", scope.Domain,
					"region", scope.Region,
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

// collectLoadBalancers collects the OpenStack LoadBalancers,
// using the client associated with the client scope in the given payload.
func collectLoadBalancers(ctx context.Context, payload CollectLoadBalancersPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.LoadBalancerClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack load balancers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	items := make([]models.LoadBalancer, 0)
	lbWithPoolItems := make([]models.LoadBalancerWithPool, 0)

	err := loadbalancers.List(client.Client, nil).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				lbList, err := loadbalancers.ExtractLoadBalancers(page)

				if err != nil {
					logger.Error(
						"could not extract load balancers pages",
						"project", payload.Scope.Project,
						"domain", payload.Scope.Domain,
						"region", payload.Scope.Region,
						"reason", err,
					)

					return false, err
				}

				for _, lb := range lbList {
					for _, pool := range lb.Pools {
						item := models.LoadBalancerWithPool{
							LoadBalancerID: lb.ID,
							PoolID:         pool.ID,
							ProjectID:      lb.ProjectID,
						}

						lbWithPoolItems = append(lbWithPoolItems, item)
					}

					item := models.LoadBalancer{
						LoadBalancerID: lb.ID,
						Name:           lb.Name,
						ProjectID:      lb.ProjectID,
						Domain:         client.Domain,
						Region:         client.Region,
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
		"populated openstack load balancers",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	if len(lbWithPoolItems) == 0 {
		return nil
	}

	out, err = db.DB.NewInsert().
		Model(&lbWithPoolItems).
		On("CONFLICT (loadbalancer_id, pool_id, project_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert load balancers with pools into db",
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
		"populated openstack load balancers with pools",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}
