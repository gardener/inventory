// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"time"

	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// Project represents a GCP Project.
type Project struct {
	bun.BaseModel `bun:"table:gcp_project"`
	coremodels.Model

	// Name is the globally unique id of the project represented as
	// "projects/<uin64>" value
	Name string `bun:"name,notnull,unique"`
	// ProjectID is the user-defined globally unique project id.
	ProjectID string `bun:"project_id,notnull,unique"`

	Parent            string    `bun:"parent,notnull"`
	State             string    `bun:"state,notnull"`
	DisplayName       string    `bun:"display_name,notnull"`
	ProjectCreateTime time.Time `bun:"project_create_time,nullzero"`
	ProjectUpdateTime time.Time `bun:"project_update_time,nullzero"`
	ProjectDeleteTime time.Time `bun:"project_delete_time,nullzero"`
	Etag              string    `bun:"etag,notnull"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("gcp:model:project", &Project{})

	// Link tables
}
