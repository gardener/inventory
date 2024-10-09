// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/azure/models"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectSubscriptions is the name of the task for collecting Azure
// Subscriptions.
const TaskCollectSubscriptions = "az:task:collect-subscriptions"

// NewCollectSubscriptionsTask creates a new [asynq.Task] for collecting Azure
// Subscriptions, without specifying a payload.
func NewCollectSubscriptionsTasks() *asynq.Task {
	return asynq.NewTask(TaskCollectSubscriptions, nil)
}

// HandleCollectSubscriptionsTask is the handler, which collects Azure
// Subscriptions.
func HandleCollectSubscriptionsTask(ctx context.Context, t *asynq.Task) error {
	logger := asynqutils.GetLogger(ctx)
	if azureclients.SubscriptionsClientset.Length() == 0 {
		logger.Warn("no Azure subscriptions clients found")
		return nil
	}

	items := make([]models.Subscription, 0)
	err := azureclients.SubscriptionsClientset.Range(func(subscriptionID string, client *azureclients.Client[*armsubscription.SubscriptionsClient]) error {
		logger.Info("collecting Azure subscription", "subscription_id", subscriptionID)
		sub, err := client.Client.Get(ctx, subscriptionID, &armsubscription.SubscriptionsClientGetOptions{})
		if err != nil {
			logger.Error(
				"failed to get Azure subscription",
				"subscription_id", subscriptionID,
				"reason", err,
			)
			return registry.ErrContinue
		}
		item := models.Subscription{
			SubscriptionID: ptr.Value(sub.SubscriptionID, ""),
			Name:           ptr.Value(sub.DisplayName, ""),
			State:          string(ptr.Value(sub.State, armsubscription.SubscriptionState(""))),
		}
		items = append(items, item)
		return nil
	})

	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (subscription_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("state = EXCLUDED.state").
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

	logger.Info("populated azure subscriptions", "count", count)

	return nil
}
