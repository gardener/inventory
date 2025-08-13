// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	gardenerutils "github.com/gardener/inventory/pkg/gardener/utils"
	"github.com/gardener/inventory/pkg/metrics"
	"github.com/gardener/inventory/pkg/openstack/models"
	openstackutils "github.com/gardener/inventory/pkg/openstack/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

const (
	// TaskCollectPools is the name of the task for collecting OpenStack
	// Pools.
	TaskCollectPools = "openstack:task:collect-pools"
	// TaskCollectPoolMembers is the name of the task for collecting OpenStack
	// Pool Members for a specific pool.
	TaskCollectPoolMembers = "openstack:task:collect-pool-members"
)

// CollectPoolsPayload represents the payload, which specifies
// where to collect OpenStack Pools from.
type CollectPoolsPayload struct {
	// Scope specifies the project scope to use for collection.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// CollectPoolMembersPayload represents the payload for collecting pool members
// for a specific pool.
type CollectPoolMembersPayload struct {
	// Scope specifies the project scope to use for collection.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
	// PoolID is the ID of the pool to collect members for.
	PoolID string `json:"pool_id" yaml:"pool_id"`
}

// NewCollectPoolsTask creates a new [asynq.Task] for collecting OpenStack
// Pools, without specifying a payload.
func NewCollectPoolsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectPools, nil)
}

// NewCollectPoolMembersTask creates a new [asynq.Task] for collecting OpenStack
// Pool Members for a specific pool.
func NewCollectPoolMembersTask(payload CollectPoolMembersPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskCollectPoolMembers, data), nil
}

// HandleCollectPoolsTask handles the task for collecting OpenStack Pools.
func HandleCollectPoolsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Pools for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectPools(ctx)
	}

	var payload CollectPoolsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectPools(ctx, payload)
}

// HandleCollectPoolMembersTask handles the task for collecting OpenStack Pool Members
// for a specific pool.
func HandleCollectPoolMembersTask(ctx context.Context, t *asynq.Task) error {
	var payload CollectPoolMembersPayload
	if err := asynqutils.Unmarshal(t.Payload(), &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	if payload.PoolID == "" {
		return asynqutils.SkipRetry(errors.New("empty pool ID specified"))
	}

	return collectPoolMembers(ctx, payload)
}

// enqueueCollectPools enqueues tasks for collecting OpenStack Pools from
// all configured OpenStack clients by creating a payload with the respective
// client scope.
func enqueueCollectPools(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.LoadBalancerClientset.Length() == 0 {
		logger.Warn("no OpenStack loadbalancer clients found")

		return nil
	}

	return openstackclients.LoadBalancerClientset.
		Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
			payload := CollectPoolsPayload{
				Scope: scope,
			}
			data, err := json.Marshal(payload)
			if err != nil {
				logger.Error(
					"failed to marshal payload for OpenStack pools",
					"project", scope.Project,
					"domain", scope.Domain,
					"region", scope.Region,
					"reason", err,
				)

				return err
			}

			task := asynq.NewTask(TaskCollectPools, data)
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

// collectPools collects the OpenStack Pools,
// using the client associated with the client scope in the given payload.
// For each pool found, it enqueues a separate task to collect pool members.
func collectPools(ctx context.Context, payload CollectPoolsPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.LoadBalancerClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack pools",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			poolsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		key := metrics.Key(
			TaskCollectPools,
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	poolItems := make([]models.Pool, 0)

	opts := pools.ListOpts{
		ProjectID: client.ProjectID,
	}
	err := pools.List(client.Client, opts).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				extractedPools, err := pools.ExtractPools(page)

				if err != nil {
					logger.Error(
						"could not extract pool pages",
						"project", payload.Scope.Project,
						"domain", payload.Scope.Domain,
						"region", payload.Scope.Region,
						"reason", err,
					)

					return false, err
				}

				for _, pool := range extractedPools {
					// Create pool record
					item := models.Pool{
						PoolID:      pool.ID,
						ProjectID:   pool.ProjectID,
						Name:        pool.Name,
						SubnetID:    pool.SubnetID,
						Description: pool.Description,
					}
					poolItems = append(poolItems, item)

					// Enqueue task to collect pool members for this pool
					memberPayload := CollectPoolMembersPayload{
						Scope:    payload.Scope,
						PoolID:   pool.ID,
					}
					data, err := json.Marshal(memberPayload)
					if err != nil {
						logger.Error(
							"failed to marshal pool member payload",
							"pool_id", pool.ID,
							"pool_name", pool.Name,
							"project", payload.Scope.Project,
							"reason", err,
						)

						continue
					}

					task := asynq.NewTask(TaskCollectPoolMembers, data)
					info, err := asynqclient.Client.Enqueue(task)
					if err != nil {
						logger.Error(
							"failed to enqueue pool member collection task",
							"pool_id", pool.ID,
							"pool_name", pool.Name,
							"project", payload.Scope.Project,
							"reason", err,
						)

						continue
					}

					logger.Info(
						"enqueued pool member collection task",
						"task_id", info.ID,
						"pool_id", pool.ID,
						"pool_name", pool.Name,
						"project", payload.Scope.Project,
					)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract pool pages",
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	if len(poolItems) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&poolItems).
		On("CONFLICT (pool_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("subnet_id = EXCLUDED.subnet_id").
		Set("description = EXCLUDED.description").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert pools into db",
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
		"populated openstack pools",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}

// collectPoolMembers collects the OpenStack Pool Members for a specific pool,
// using the client associated with the client scope in the given payload.
func collectPoolMembers(ctx context.Context, payload CollectPoolMembersPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.LoadBalancerClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack pool members",
		"pool_id", payload.PoolID,
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	var memberCount int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			poolMembersDesc,
			prometheus.GaugeValue,
			float64(memberCount),
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
			payload.PoolID,
		)
		key := metrics.Key(
			TaskCollectPoolMembers,
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
			payload.PoolID,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	memberItems := make([]models.PoolMember, 0)

	memberOpts := pools.ListMembersOpts{
		ProjectID: client.ProjectID,
	}
	err := pools.ListMembers(client.Client, payload.PoolID, memberOpts).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
				extractedMembers, err := pools.ExtractMembers(page)

				if err != nil {
					logger.Error(
						"could not extract pool member pages",
						"pool_id", payload.PoolID,
						"project", payload.Scope.Project,
						"domain", payload.Scope.Domain,
						"region", payload.Scope.Region,
						"reason", err,
					)

					return false, err
				}

				for _, member := range extractedMembers {
					var inferredGardenerShoot string
					shoot, err := gardenerutils.InferShootFromInstanceName(ctx, member.Name)
					if err == nil {
						inferredGardenerShoot = shoot.TechnicalID
					}

					item := models.PoolMember{
						MemberID:              member.ID,
						PoolID:                payload.PoolID,
						ProjectID:             member.ProjectID,
						Name:                  member.Name,
						InferredGardenerShoot: inferredGardenerShoot,
						SubnetID:              member.SubnetID,
						ProtocolPort:          member.ProtocolPort,
						MemberCreatedAt:       member.CreatedAt,
						MemberUpdatedAt:       member.UpdatedAt,
					}

					memberItems = append(memberItems, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract pool member pages",
			"pool_id", payload.PoolID,
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	if len(memberItems) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&memberItems).
		On("CONFLICT (member_id, pool_id, project_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("subnet_id = EXCLUDED.subnet_id").
		Set("protocol_port = EXCLUDED.protocol_port").
		Set("inferred_gardener_shoot = EXCLUDED.inferred_gardener_shoot").
		Set("member_created_at = EXCLUDED.member_created_at").
		Set("member_updated_at = EXCLUDED.member_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert pool members into db",
			"pool_id", payload.PoolID,
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	memberCount, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack pool members",
		"pool_id", payload.PoolID,
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", memberCount,
	)

	return nil
}
