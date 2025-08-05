// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
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
	// TaskCollectVolumes is the name of the task for collecting OpenStack
	// Volumes.
	TaskCollectVolumes = "openstack:task:collect-volumes"
)

// CollectVolumesPayload represents the payload, which specifies
// where to collect OpenStack Volumes from.
type CollectVolumesPayload struct {
	// Scope specifies the client scope for which to collect.
	Scope openstackclients.ClientScope `json:"scope" yaml:"scope"`
}

// NewCollectVolumesTask creates a new [asynq.Task] for collecting OpenStack
// Volumes, without specifying a payload.
func NewCollectVolumesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectVolumes, nil)
}

// HandleCollectVolumesTask handles the task for collecting OpenStack Volumes.
func HandleCollectVolumesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting OpenStack Volumes from all configured volume clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectVolumes(ctx)
	}

	var payload CollectVolumesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if err := openstackutils.IsValidProjectScope(payload.Scope); err != nil {
		return asynqutils.SkipRetry(ErrInvalidScope)
	}

	return collectVolumes(ctx, payload)
}

// enqueueCollectVolumes enqueues tasks for collecting OpenStack Volumes from
// all configured OpenStack volume clients by creating a payload with the respective
// client scope.
func enqueueCollectVolumes(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	if openstackclients.BlockStorageClientset.Length() == 0 {
		logger.Warn("no OpenStack blockstorage clients found")

		return nil
	}

	queue := asynqutils.GetQueueName(ctx)

	return openstackclients.BlockStorageClientset.Range(func(scope openstackclients.ClientScope, _ openstackclients.Client[*gophercloud.ServiceClient]) error {
		payload := CollectVolumesPayload{
			Scope: scope,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for OpenStack volumes",
				"project", scope.Project,
				"domain", scope.Domain,
				"region", scope.Region,
				"reason", err,
			)

			return err
		}

		task := asynq.NewTask(TaskCollectVolumes, data)
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

// collectVolumes collects the OpenStack Volumes,
// using the client associated with the client scope in the given payload.
func collectVolumes(ctx context.Context, payload CollectVolumesPayload) error {
	logger := asynqutils.GetLogger(ctx)

	client, ok := openstackclients.BlockStorageClientset.Get(payload.Scope)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.Scope.Project))
	}

	logger.Info(
		"collecting OpenStack volumes",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			volumesDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		key := metrics.Key(
			TaskCollectVolumes,
			payload.Scope.Project,
			payload.Scope.Domain,
			payload.Scope.Region,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	items := make([]models.Volume, 0)

	opts := volumes.ListOpts{
		TenantID: client.ProjectID,
	}
	err := volumes.List(client.Client, opts).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				volumeList, err := volumes.ExtractVolumes(page)

				if err != nil {
					logger.Error(
						"could not extract volume pages",
						"reason", err,
					)

					return false, err
				}

				for _, v := range volumeList {
					item := models.Volume{
						Name:              v.Name,
						VolumeID:          v.ID,
						ProjectID:         v.TenantID,
						Domain:            client.Domain,
						Region:            client.Region,
						UserID:            v.UserID,
						AvailabilityZone:  v.AvailabilityZone,
						Size:              v.Size,
						VolumeType:        v.VolumeType,
						Status:            v.Status,
						ReplicationStatus: v.ReplicationStatus,
						Bootable:          v.Bootable,
						Encrypted:         v.Encrypted,
						MultiAttach:       v.Multiattach,
						SnapshotID:        v.SnapshotID,
						Description:       v.Description,
						TimeCreated:       v.CreatedAt,
						TimeUpdated:       v.UpdatedAt,
					}

					items = append(items, item)
				}

				return true, nil
			})

	if err != nil {
		logger.Error(
			"could not extract volume pages",
			"reason", err,
		)

		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (volume_id, project_id, domain, region) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("user_id = EXCLUDED.user_id").
		Set("availability_zone = EXCLUDED.availability_zone").
		Set("size = EXCLUDED.size").
		Set("volume_type = EXCLUDED.volume_type").
		Set("status = EXCLUDED.status").
		Set("replication_status = EXCLUDED.replication_status").
		Set("bootable = EXCLUDED.bootable").
		Set("encrypted = EXCLUDED.encrypted").
		Set("multi_attach = EXCLUDED.multi_attach").
		Set("snapshot_id = EXCLUDED.snapshot_id").
		Set("description = EXCLUDED.description").
		Set("volume_created_at = EXCLUDED.volume_created_at").
		Set("volume_updated_at = EXCLUDED.volume_updated_at").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert volumes into db",
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
		"populated openstack volumes",
		"project", payload.Scope.Project,
		"domain", payload.Scope.Domain,
		"region", payload.Scope.Region,
		"count", count,
	)

	return nil
}
