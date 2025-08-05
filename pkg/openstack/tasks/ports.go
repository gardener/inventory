// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"net"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
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
)

const (
	// TaskCollectPorts is the name of the task for collecting OpenStack Ports.
	TaskCollectPorts = "openstack:task:collect-ports"
)

// CollectPortsPayload represents the payload, which specifies
// where to collect OpenStack Ports from.
type CollectPortsPayload struct {
	// Scope specifies the client scope for which to collect.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectPortsTask creates a new [asynq.Task] for collecting OpenStack
// Ports, without specifying a payload.
func NewCollectPortsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectPorts, nil)
}

// HandleCollectPortsTask handles the task for collecting OpenStack Ports.
func HandleCollectPortsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Ports for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectPorts(ctx)
	}

	var payload CollectPortsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(err)
	}

	return collectPorts(ctx, payload)
}

// enqueueCollectPorts enqueues tasks for collecting OpenStack Ports from
// all configured OpenStack network clients by creating a payload with the respective
// client scope.
func enqueueCollectPorts(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.NetworkClientset.Length() == 0 {
		logger.Warn("no OpenStack network clients found")

		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.NetworkClientset.Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectPortsPayload{
			Scope: scope,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack ports",
				"project", scope.Project,
				"domain", scope.Domain,
				"region", scope.Region,
				"reason", err,
			)

			return err
		}

		task := asynq.NewTask(TaskCollectPorts, data)
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

// collectPorts collects the OpenStack Ports,
// using the client associated with the client scope in the given payload.
func collectPorts(ctx context.Context, payload CollectPortsPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.NetworkClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack ports",
		"project", payload.Scope.Project,
		"region", payload.Scope.Region,
		"domain", payload.Scope.Domain,
	)

	var portCount int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			portsDesc,
			prometheus.GaugeValue,
			float64(portCount),
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		key := metrics.Key(
			TaskCollectPorts,
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	items := make([]models.Port, 0)
	portIPs := make([]models.PortIP, 0)

	opts := ports.ListOpts{
		ProjectID: client.ClientScope.ProjectID,
	}
	err := ports.List(client.Client, opts).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				portList, err := ports.ExtractPorts(page)
				if err != nil {
					logger.Error("failed to extract ports", "reason", err)

					return false, err
				}

				for _, port := range portList {
					items = append(items, models.Port{
						PortID:      port.ID,
						Name:        port.Name,
						ProjectID:   port.ProjectID,
						Domain:      payload.Scope.Domain,
						Region:      payload.Scope.Region,
						NetworkID:   port.NetworkID,
						DeviceID:    port.DeviceID,
						DeviceOwner: port.DeviceOwner,
						MacAddress:  port.MACAddress,
						Status:      port.Status,
						Description: port.Description,
						TimeCreated: port.CreatedAt,
						TimeUpdated: port.UpdatedAt,
					})

					for _, fixedIP := range port.FixedIPs {
						parsedIP := net.ParseIP(fixedIP.IPAddress)
						if parsedIP == nil {
							logger.Warn(
								"unable to parse IP",
								"ip string value",
								fixedIP.IPAddress,
							)

							continue
						}

						ip := models.PortIP{
							PortID:    port.ID,
							IPAddress: parsedIP,
							SubnetID:  fixedIP.SubnetID,
						}
						portIPs = append(portIPs, ip)
					}
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"failed to collect OpenStack ports",
			"project", payload.Scope.Project,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (port_id, project_id, network_id, region) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("domain = EXCLUDED.domain").
		Set("device_id = EXCLUDED.device_id").
		Set("device_owner = EXCLUDED.device_owner").
		Set("mac_address = EXCLUDED.mac_address").
		Set("status = EXCLUDED.status").
		Set("description = EXCLUDED.description").
		Set("port_created_at = EXCLUDED.port_created_at").
		Set("port_updated_at = EXCLUDED.port_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert ports into db",
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	portCount, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack ports",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", portCount,
	)

	if len(portIPs) == 0 {
		return nil
	}

	out, err = db.DB.NewInsert().
		Model(&portIPs).
		On("CONFLICT (port_id, ip_address, subnet_id, project_id) DO UPDATE").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert port IPs into db",
			"project", payload.Scope.Project,
			"domain", payload.Scope.Domain,
			"region", payload.Scope.Region,
			"reason", err,
		)

		return err
	}

	ipCount, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated openstack port IPs",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", ipCount,
	)

	return nil
}
