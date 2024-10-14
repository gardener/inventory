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

// LinkVirtualMachineWithResourceGroup creates links between the
// [models.VirtualMachine] and [models.ResourceGroup] models.
func LinkVirtualMachineWithResourceGroup(ctx context.Context, db *bun.DB) error {
	var items []models.VirtualMachine
	err := db.NewSelect().
		Model(&items).
		Relation("ResourceGroup").
		Where("resource_group.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.VirtualMachineToResourceGroup, 0, len(items))
	for _, item := range items {
		link := models.VirtualMachineToResourceGroup{
			VMID:            item.ID,
			ResourceGroupID: item.ResourceGroup.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (rg_id, vm_id) DO UPDATE").
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
	logger.Info("linked azure vm with resource group", "count", count)

	return nil
}

// LinkPublicAddressWithResourceGroup establishes relationships between the
// [models.PublicAddress] and [models.ResourceGroup] models.
func LinkPublicAddressWithResourceGroup(ctx context.Context, db *bun.DB) error {
	var items []models.PublicAddress
	err := db.NewSelect().
		Model(&items).
		Relation("ResourceGroup").
		Where("resource_group.id IS NOT NULL").
		Scan(ctx)

	if err != nil {
		return err
	}

	links := make([]models.PublicAddressToResourceGroup, 0, len(items))
	for _, item := range items {
		link := models.PublicAddressToResourceGroup{
			PublicAddressID: item.ID,
			ResourceGroupID: item.ResourceGroup.ID,
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return nil
	}

	out, err := db.NewInsert().
		Model(&links).
		On("CONFLICT (rg_id, pa_id) DO UPDATE").
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
	logger.Info("linked azure public address with resource group", "count", count)

	return nil
}
