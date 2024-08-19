// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/gardener/models"
)

// GetSeedsFromDB fetches the [models.Seed] items from the database.
func GetSeedsFromDB(ctx context.Context) ([]models.Seed, error) {
	items := make([]models.Seed, 0)
	err := db.DB.NewSelect().Model(&items).Scan(ctx)
	return items, err
}
