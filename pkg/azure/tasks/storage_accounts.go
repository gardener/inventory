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

	"github.com/gardener/inventory/pkg/azure/models"
	azureutils "github.com/gardener/inventory/pkg/azure/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectStorageAccounts is the name of the task for collecting Azure Storage Accounts.
const TaskCollectStorageAccounts = "az:task:collect-storage-accounts"

// CollectStorageAccountsPayload is the payload used for collecting Azure
// Storage Accounts.
type CollectStorageAccountsPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`
}

// NewCollectStorageAccountsTask creates a new [asynq.Task] for collecting Azure
// Storage Accounts without specifying a payload.
func NewCollectStorageAccountsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectStorageAccounts, nil)
}

// HandleCollectStorageAccountsTask is the handler, which collects Azure
// Storage Accounts.
func HandleCollectStorageAccountsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known resource groups.
	data := t.Payload()
	if data == nil {
		return enqueueCollectStorageAccounts(ctx)
	}

	var payload CollectStorageAccountsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}

	return collectStorageAccounts(ctx, payload)
}

// enqueueCollectStorageAccounts enqueues tasks for collecting Azure Storage Accounts for known Resource Groups.
func enqueueCollectStorageAccounts(ctx context.Context) error {
	resourceGroups, err := azureutils.GetResourceGroupsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)
	for _, rg := range resourceGroups {
		if !azureclients.StorageAccountsClientset.Exists(rg.SubscriptionID) {
			logger.Warn(
				"Azure Storage Accounts client not found",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
			)

			continue
		}

		payload := CollectStorageAccountsPayload{
			SubscriptionID: rg.SubscriptionID,
			ResourceGroup:  rg.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure Storage Accounts",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
				"reason", err,
			)

			continue
		}
		task := asynq.NewTask(TaskCollectStorageAccounts, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"subscription_id", rg.SubscriptionID,
			"resource_group", rg.Name,
		)
	}

	return nil
}

// collectStorageAccounts collects the Azure Storage Accounts from the
// subscription and resource group specified in the payload.
func collectStorageAccounts(ctx context.Context, payload CollectStorageAccountsPayload) error {
	client, ok := azureclients.StorageAccountsClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure Storage Accounts",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
	)

	items := make([]models.StorageAccount, 0)
	pager := client.Client.NewListByResourceGroupPager(
		payload.ResourceGroup,
		&armstorage.AccountsClientListByResourceGroupOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure Storage Accounts",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"reason", err,
			)

			return azureutils.MaybeSkipRetry(err)
		}

		for _, account := range page.Value {
			var provisioningState armstorage.ProvisioningState
			var skuName armstorage.SKUName
			var skuTier armstorage.SKUTier
			var kind armstorage.Kind
			var creationTime time.Time

			if account.SKU != nil {
				skuName = ptr.Value(account.SKU.Name, armstorage.SKUName(""))
				skuTier = ptr.Value(account.SKU.Tier, armstorage.SKUTier(""))
			}

			kind = ptr.Value(account.Kind, armstorage.Kind(""))
			if account.Properties != nil {
				provisioningState = ptr.Value(account.Properties.ProvisioningState, armstorage.ProvisioningState(""))
				creationTime = ptr.Value(account.Properties.CreationTime, time.Time{})
			}

			item := models.StorageAccount{
				Name:              ptr.Value(account.Name, ""),
				SubscriptionID:    payload.SubscriptionID,
				ResourceGroupName: payload.ResourceGroup,
				Location:          ptr.Value(account.Location, ""),
				ProvisioningState: string(provisioningState),
				Kind:              string(kind),
				SKUName:           string(skuName),
				SKUTier:           string(skuTier),
				CreationTime:      creationTime,
			}
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, resource_group, subscription_id) DO UPDATE").
		Set("location = EXCLUDED.location").
		Set("provisioning_state = EXCLUDED.provisioning_state").
		Set("kind = EXCLUDED.kind").
		Set("sku_name = EXCLUDED.sku_name").
		Set("sku_tier = EXCLUDED.sku_tier").
		Set("creation_time = EXCLUDED.creation_time").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated azure storage accounts", "count", count)

	return nil
}
