// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"

	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients/db"
)

// GetRegionsFromDB gets the AWS Regions from the database.
func GetRegionsFromDB(ctx context.Context) ([]models.Region, error) {
	items := make([]models.Region, 0)
	err := db.DB.NewSelect().Model(&items).Scan(ctx)
	return items, err
}
