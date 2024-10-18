// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/azure/models"
	azureutils "github.com/gardener/inventory/pkg/azure/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectResourceGroups is the name of the task for collecting Azure
// Resource Groups.
const TaskCollectResourceGroups = "az:task:collect-resource-groups"

// CollectResourceGroupsPayload is the payload used for collecting Azure
// Resource Groups.
type CollectResourceGroupsPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`
}

// NewCollectResourceGroups creates a new [asynq.Task] for collecting Azure
// Resource Groups, without specifying a payload.
func NewCollectResourceGroupsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectResourceGroups, nil)
}

// HandleCollectResourceGroupsTask is the handler, which collects Azure
// Resource Groups.
func HandleCollectResourceGroupsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known subscriptions.
	data := t.Payload()
	if data == nil {
		return enqueueCollectResourceGroups(ctx)
	}

	var payload CollectResourceGroupsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}

	return collectResourceGroups(ctx, payload)
}

// enqueueCollectResourceGroups enqueues tasks for collecting Azure Resource
// Groups for all known Subscriptions.
func enqueueCollectResourceGroups(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if azureclients.ResourceGroupsClientset.Length() == 0 {
		logger.Warn("no Azure Resource Groups clients found")
		return nil
	}

	err := azureclients.ResourceGroupsClientset.Range(func(subscriptionID string, _ *azureclients.Client[*armresources.ResourceGroupsClient]) error {
		payload := CollectResourceGroupsPayload{
			SubscriptionID: subscriptionID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure Resource Groups",
				"subscription_id", subscriptionID,
				"reason", err,
			)
			return registry.ErrContinue
		}
		task := asynq.NewTask(TaskCollectResourceGroups, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"subscription_id", subscriptionID,
				"reason", err,
			)
			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"subscription_id", subscriptionID,
		)
		return nil
	})

	return err
}

// collectResourceGroups collects the Azure Resource Groups from the
// subscription specified in the payload.
func collectResourceGroups(ctx context.Context, payload CollectResourceGroupsPayload) error {
	client, ok := azureclients.ResourceGroupsClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting Azure Resource Groups", "subscription_id", payload.SubscriptionID)

	items := make([]models.ResourceGroup, 0)
	pager := client.Client.NewListPager(&armresources.ResourceGroupsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure Resource Groups",
				"subscription_id", payload.SubscriptionID,
				"reason", err,
			)
			return azureutils.MaybeSkipRetry(err)
		}
		for _, rg := range page.Value {
			item := models.ResourceGroup{
				Name:           ptr.Value(rg.Name, ""),
				Location:       ptr.Value(rg.Location, ""),
				SubscriptionID: payload.SubscriptionID,
			}
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (subscription_id, name) DO UPDATE").
		Set("location = EXCLUDED.location").
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

	logger.Info("populated azure resource groups", "count", count)

	return nil
}
