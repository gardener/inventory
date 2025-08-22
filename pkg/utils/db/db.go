// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/gardener/inventory/pkg/core/config"
	dbclient "github.com/gardener/inventory/pkg/clients/db"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// ErrInvalidDSN error is returned, when the DSN configuration is incorrect, or
// empty.
var ErrInvalidDSN = errors.New("invalid or missing database configuration")

// NewFromConfig creates a new [bun.DB] based on the provided
// [config.DatabaseConfig] spec.
func NewFromConfig(conf config.DatabaseConfig) (*bun.DB, error) {
	if conf.DSN == "" {
		return nil, ErrInvalidDSN
	}

	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(conf.DSN)))
	db := bun.NewDB(pgdb, pgdialect.New())

	return db, nil
}

// LinkFunction is a function, which establishes relationships between models.
type LinkFunction func(ctx context.Context, db *bun.DB) error

// LinkObjects links objects by using the provided [LinkFunction] items.
func LinkObjects(ctx context.Context, db *bun.DB, items []LinkFunction) error {
	logger := asynqutils.GetLogger(ctx)
	for _, linkFunc := range items {
		if err := linkFunc(ctx, db); err != nil {
			logger.Error("failed to link objects", "reason", err)

			continue
		}
	}

	return nil
}

// GetResourcesFromDB fetches the given model from the database.
func GetResourcesFromDB[T any](ctx context.Context) ([]T, error) {
	items := make([]T, 0)
	err := dbclient.DB.NewSelect().Model(&items).Scan(ctx)

	return items, err
}
