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

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("az:model:subscription", &Subscription{})

	// Link tables
}
