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

	ServerID         string    `bun:"server_id,unique:openstack_server_key"`
	Name             string    `bun:"name,notnull"`
	ProjectID        string    `bun:"project_id,unique:openstack_server_key"`
	Domain           string    `bun:"domain,notnull"`
	Region           string    `bun:"region,notnull"`
	UserID           string    `bun:"user_id,nullzero"`
	AvailabilityZone string    `bun:"availability_zone,nullzero"`
	Status           string    `bun:"status,nullzero"`
	ImageID          string    `bun:"image_id,nullzero"`
	TimeCreated      time.Time `bun:"server_created_at,nullzero"`
	TimeUpdated      time.Time `bun:"server_updated_at,nullzero"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("openstack:model:server", &Server{})
}
