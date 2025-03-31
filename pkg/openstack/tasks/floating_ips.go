// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"net"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
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
	// TaskCollectFloatingIPs is the name of the task for collecting OpenStack
	// Floating IPs.
	TaskCollectFloatingIPs = "openstack:task:collect-floating-ips"
)

// CollectFloatingIPsPayload represents the payload, which specifies
// the scope for collecting OpenStack Floating IPs.
type CollectFloatingIPsPayload struct {
	// Scope specifies the client scope to use for collection.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectFloatingIPsTask creates a new [asynq.Task] for collecting OpenStack
// FloatingIPs, without specifying a payload.
func NewCollectFloatingIPsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectFloatingIPs, nil)
}

// HandleCollectFloatingIPsTask handles the task for collecting OpenStack FloatingIPs.
func HandleCollectFloatingIPsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Floating IPs for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectFloatingIPs(ctx)
	}

	var payload CollectFloatingIPsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectFloatingIPs(ctx, payload)

}

// enqueueCollectFloatingIPs enqueues tasks for collecting OpenStack Floating IPs for
// all configured OpenStack network clients by creating a payload with the respective
// client scope.
func enqueueCollectFloatingIPs(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.NetworkClientset.Length() == 0 {
		logger.Warn("no OpenStack network clients found")
		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.NetworkClientset.Range(func(scope openstackclients.ClientScope, client openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectFloatingIPsPayload{
			Scope: scope,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack floating IPs",
				"scope", scope,
				"reason", err,
			)
			return err
		}

		task := asynq.NewTask(TaskCollectFloatingIPs, data)
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

// collectFloatingIPs collects the OpenStack Floating IPs,
// using the client associated with the client scope in the given payload.
func collectFloatingIPs(ctx context.Context, payload CollectFloatingIPsPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.NetworkClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack floating IPs",
		"scope", payload.Scope,
	)

	items := make([]models.FloatingIP, 0)

	err := floatingips.List(client.Client, nil).
		EachPage(ctx,
			func(ctx context.Context, page pagination.Page) (bool, error) {
				floatingIPList, err := floatingips.ExtractFloatingIPs(page)

				if err != nil {
					logger.Error(
						"could not extract floating IPs pages",
						"reason", err,
					)
					return false, err
				}

				for _, ip := range floatingIPList {
					fixedIP := net.ParseIP(ip.FixedIP)
					floatingIP := net.ParseIP(ip.FloatingIP)

					if fixedIP == nil {
						logger.Warn(
							"Invalid fixed IP provided",
							"fixed IP",
							ip.FixedIP,
						)
						continue
					}

					if floatingIP == nil {
						logger.Warn(
							"Invalid floating IP provided",
							"floating IP",
							ip.FloatingIP,
						)
						continue
					}

					item := models.FloatingIP{
						FloatingIPID:      ip.ID,
						ProjectID:         ip.TenantID,
						Domain:            client.Domain,
						Region:            client.Region,
						PortID:            ip.PortID,
						FixedIP:           fixedIP,
						RouterID:          ip.RouterID,
						FloatingIP:        floatingIP,
						FloatingNetworkID: ip.FloatingNetworkID,
						Description:       ip.Description,
						TimeCreated:       ip.CreatedAt,
						TimeUpdated:       ip.UpdatedAt,
					}
					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract floating IP pages",
			"reason", err,
		)
		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (floating_ip_id, project_id) DO UPDATE").
		Set("domain = EXCLUDED.domain").
		Set("region = EXCLUDED.region").
		Set("port_id = EXCLUDED.port_id").
		Set("fixed_ip = EXCLUDED.fixed_ip").
		Set("router_id = EXCLUDED.router_id").
		Set("floating_ip = EXCLUDED.floating_ip").
		Set("floating_network_id = EXCLUDED.floating_network_id").
		Set("description = EXCLUDED.description").
		Set("ip_created_at = EXCLUDED.ip_created_at").
		Set("ip_updated_at = EXCLUDED.ip_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert floating IPs into db",
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
		"populated openstack floating IPs",
		"scope", payload.Scope,
		"count", count,
	)

	return nil
}
