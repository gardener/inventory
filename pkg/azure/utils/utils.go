// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/gardener/inventory/pkg/azure/models"
	"github.com/gardener/inventory/pkg/clients/db"
)

// GetResourceGroupsFromDB returns the [models.ResourceGroup] from the database.
func GetResourceGroupsFromDB(ctx context.Context) ([]models.ResourceGroup, error) {
	items := make([]models.ResourceGroup, 0)
	err := db.DB.NewSelect().Model(&items).Scan(ctx)
	return items, err
}
