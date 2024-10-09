// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// Subscription represents an Azure Subscription
type Subscription struct {
	bun.BaseModel `bun:"table:az_subscription"`
	coremodels.Model

	SubscriptionID string `bun:"subscription_id,notnull,unique"`
	Name           string `bun:"name,nullzero"`
	State          string `bun:"state,nullzero"`
}

// ResourceGroup represents an Azure Resource Group
type ResourceGroup struct {
	bun.BaseModel `bun:"table:az_resource_group"`
	coremodels.Model

	Name           string `bun:"name,notnull,unique:az_resource_group_key"`
	SubscriptionID string `bun:"subscription_id,notnull,unique:az_resource_group_key"`
	Location       string `bun:"location,notnull"`
}

// ResourceGroupToSubscription represents a link table connecting the
// [Subscription] with [ResourceGroup] models.
type ResourceGroupToSubscription struct {
	bun.BaseModel `bun:"table:l_az_rg_to_subscription"`
	coremodels.Model

	ResourceGroupID uint64 `bun:"rg_id,notnull,unique:l_az_rg_to_subscription_key"`
	SubscriptionID  uint64 `bun:"sub_id,notnull,unique:l_az_rg_to_subscription_key"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("az:model:subscription", &Subscription{})
	registry.ModelRegistry.MustRegister("az:model:resource_group", &ResourceGroup{})

	// Link tables
	registry.ModelRegistry.MustRegister("az:model:link_rg_to_subscription", &ResourceGroupToSubscription{})
}
