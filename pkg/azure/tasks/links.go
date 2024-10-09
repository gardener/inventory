// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/gardener/inventory/pkg/azure/models"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// LinkResourceGroupWithSubscription creates links between the
// [models.ResourceGroup] and [models.Subscription] models.
func LinkResourceGroupWithSubscription(ctx context.Context, db *bun.DB) error {
	var items []models.ResourceGroup
	err := db.NewSelect().
		Model(&items).
		Relation("Subscription").
		Where("subscription.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.ResourceGroupToSubscription, 0, len(items))
	for _, item := range items {
		link := models.ResourceGroupToSubscription{
			ResourceGroupID: item.ID,
			SubscriptionID:  item.Subscription.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (rg_id, sub_id) DO UPDATE").
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

	logger := asynqutils.GetLogger(ctx)
	logger.Info("linked azure resource group with subscription", "count", count)

	return nil
}
