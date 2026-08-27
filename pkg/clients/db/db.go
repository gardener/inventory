// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package db

import "github.com/uptrace/bun"

// DB provides the connection to the Inventory database.
var DB *bun.DB

// SetDB sets the database connection to be used by the workers.
func SetDB(database *bun.DB) {
	DB = database
}
