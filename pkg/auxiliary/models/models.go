// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"time"

	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// HousekeeperRun represents a single run of the housekeeper.
type HousekeeperRun struct {
	bun.BaseModel `bun:"table:aux_housekeeper_run"`
	coremodels.Model

	// ModelName specifies the name of the model processed by the
	// housekeeper.
	ModelName string `bun:"model_name,notnull"`

	// StartedAt specifies when the housekeeper started processing stale
	// records.
	StartedAt time.Time `bun:"started_at,notnull"`

	// CompletedAt specifies when the housekeeper completed processing stale
	// records.
	CompletedAt time.Time `bun:"completed_at,notnull"`

	// Count specifies the number of stale records that were cleaned up by
	// the housekeeper.
	Count int64 `bun:"count,notnull"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("aux:model:housekeeper_run", &HousekeeperRun{})
}
