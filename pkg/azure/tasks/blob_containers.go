// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/azure/models"
	azureutils "github.com/gardener/inventory/pkg/azure/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectBlobContainers is the name of the task for collecting Azure Blob containers.
const TaskCollectBlobContainers = "az:task:collect-blob-containers"

// CollectBlobContainersPayload is the payload used for collecting Azure
// Blob containers.
type CollectBlobContainersPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`

	// StorageAccount specifies from which storage account to collect.
	StorageAccount string `json:"storage_account" yaml:"storage_account"`
}

// NewCollectBlobContainersTask creates a new [asynq.Task] for collecting Azure
// Blob containers without specifying a payload.
func NewCollectBlobContainersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectBlobContainers, nil)
}

// HandleCollectBlobContainersTask is the handler, which collects Azure
// Blob containers.
func HandleCollectBlobContainersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known resource groups.
	data := t.Payload()
	if data == nil {
		return enqueueCollectBlobContainers(ctx)
	}

	var payload CollectBlobContainersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}
	if payload.StorageAccount == "" {
		return asynqutils.SkipRetry(ErrNoStorageAccount)
	}

	return collectBlobContainers(ctx, payload)
}

// enqueueCollectBlobContainers enqueues tasks for collecting Azure Blob
// containers for known Resource Groups.
func enqueueCollectBlobContainers(ctx context.Context) error {
	storageAccounts, err := azureutils.GetStorageAccountsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)
	for _, acc := range storageAccounts {
		if !azureclients.BlobContainersClientset.Exists(acc.SubscriptionID) {
			logger.Warn(
				"Azure Blob containers client not found",
				"subscription_id", acc.SubscriptionID,
				"resource_group", acc.ResourceGroupName,
				"storage_account", acc.Name,
			)

			continue
		}

		payload := CollectBlobContainersPayload{
			SubscriptionID: acc.SubscriptionID,
			ResourceGroup:  acc.ResourceGroupName,
			StorageAccount: acc.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure Blob containers",
				"subscription_id", acc.SubscriptionID,
				"resource_group", acc.ResourceGroupName,
				"storage_account", acc.Name,
				"reason", err,
			)

			continue
		}
		task := asynq.NewTask(TaskCollectBlobContainers, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"subscription_id", acc.SubscriptionID,
				"resource_group", acc.ResourceGroupName,
				"storage_account", acc.Name,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"subscription_id", acc.SubscriptionID,
			"resource_group", acc.ResourceGroupName,
			"storage_account", acc.Name,
		)
	}

	return nil
}

// collectBlobContainers collects the Azure Blob containers from the
// subscription and resource group specified in the payload.
func collectBlobContainers(ctx context.Context, payload CollectBlobContainersPayload) error {
	client, ok := azureclients.BlobContainersClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure Blob containers",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
		"storage_account", payload.StorageAccount,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			blobContainersDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.SubscriptionID,
			payload.ResourceGroup,
			payload.StorageAccount,
		)
		key := metrics.Key(
			TaskCollectBlobContainers,
			payload.SubscriptionID,
			payload.ResourceGroup,
			payload.StorageAccount,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	items := make([]models.BlobContainer, 0)
	pager := client.Client.NewListPager(
		payload.ResourceGroup,
		payload.StorageAccount,
		&armstorage.BlobContainersClientListOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure Blob containers",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"storage_account", payload.StorageAccount,
				"reason", err,
			)

			return azureutils.MaybeSkipRetry(err)
		}

		for _, container := range page.Value {
			var publicAccess armstorage.PublicAccess
			var deleted bool
			var lastModifiedTime time.Time

			if container.Properties != nil {
				publicAccess = ptr.Value(container.Properties.PublicAccess, armstorage.PublicAccess(""))
				deleted = ptr.Value(container.Properties.Deleted, false)
				lastModifiedTime = ptr.Value(container.Properties.LastModifiedTime, time.Time{})
			}

			item := models.BlobContainer{
				Name:               ptr.Value(container.Name, ""),
				SubscriptionID:     payload.SubscriptionID,
				ResourceGroupName:  payload.ResourceGroup,
				StorageAccountName: payload.StorageAccount,
				PublicAccess:       string(publicAccess),
				Deleted:            deleted,
				LastModifiedTime:   lastModifiedTime,
			}
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, storage_account, resource_group, subscription_id) DO UPDATE").
		Set("public_access = EXCLUDED.public_access").
		Set("deleted = EXCLUDED.deleted").
		Set("last_modified_time = EXCLUDED.last_modified_time").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated azure blob containers", "count", count)

	return nil
}
