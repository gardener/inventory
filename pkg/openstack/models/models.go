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

// Server represents an OpenStack Server.
type Server struct {
	bun.BaseModel `bun:"table:openstack_server"`
	coremodels.Model

	ServerID         string    `bun:"server_id"`
	Name             string    `bun:"name,notnull,unique"`
	ProjectID        string    `bun:"project_id,notnull,unique"`
	Domain           string    `bun:"domain,notnull"`
	Region           string    `bun:"region,notnull"`
	UserID           string    `bun:"user_id"`
	AvailabilityZone string    `bun:"availability_zone"`
	Status           string    `bun:"status"`
	ImageID          string    `bun:"image_id"`
	TimeCreated      time.Time `bun:"server_created_at"`
	TimeUpdated      time.Time `bun:"server_updated_at"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("openstack:model:server", &Server{})
}
