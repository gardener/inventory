// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"database/sql"
	"errors"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/gardener/inventory/pkg/core/config"
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
