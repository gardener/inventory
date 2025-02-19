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

	ServerID         string    `bun:"server_id,notnull,unique:openstack_server_key"`
	Name             string    `bun:"name,notnull"`
	ProjectID        string    `bun:"project_id,notnull,unique:openstack_server_key"`
	Domain           string    `bun:"domain,notnull"`
	Region           string    `bun:"region,notnull"`
	UserID           string    `bun:"user_id,notnull"`
	AvailabilityZone string    `bun:"availability_zone,notnull"`
	Status           string    `bun:"status,notnull"`
	ImageID          string    `bun:"image_id,notnull"`
	TimeCreated      time.Time `bun:"server_created_at,notnull"`
	TimeUpdated      time.Time `bun:"server_updated_at,notnull"`
}

// Network represents an OpenStack Network.
type Network struct {
	bun.BaseModel `bun:"table:openstack_network"`
	coremodels.Model

	NetworkID   string    `bun:"network_id,notnull,unique:openstack_network_key"`
	Name        string    `bun:"name,notnull"`
	ProjectID   string    `bun:"project_id,notnull,unique:openstack_network_key"`
	Domain      string    `bun:"domain,notnull"`
	Region      string    `bun:"region,notnull"`
	Status      string    `bun:"status,notnull"`
	Shared      bool      `bun:"shared,notnull"`
	Description string    `bun:"description,notnull"`
	TimeCreated time.Time `bun:"network_created_at,notnull"`
	TimeUpdated time.Time `bun:"network_updated_at,notnull"`
}

// LoadBalancer represents an OpenStack LoadBalancer.
type LoadBalancer struct {
	bun.BaseModel `bun:"table:openstack_loadbalancer"`
	coremodels.Model

	LoadBalancerID string    `bun:"loadbalancer_id,notnull,unique:openstack_loadbalancer_key"`
	Name           string    `bun:"name,notnull"`
	ProjectID      string    `bun:"project_id,notnull,unique:openstack_loadbalancer_key"`
	Domain         string    `bun:"domain,notnull"`
	Region         string    `bun:"region,notnull"`
	Status         string    `bun:"status,notnull"`
	Provider       string    `bun:"provider,notnull"`
	VipAddress     string    `bun:"vip_address,notnull"`
	VipNetworkID   string    `bun:"vip_network_id,notnull"`
	VipSubnetID    string    `bun:"vip_subnet_id,notnull"`
	Description    string    `bun:"description,notnull"`
	TimeCreated    time.Time `bun:"loadbalancer_created_at,notnull"`
	TimeUpdated    time.Time `bun:"loadbalancer_updated_at,notnull"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("openstack:model:server", &Server{})
	registry.ModelRegistry.MustRegister("openstack:model:network", &Network{})
	registry.ModelRegistry.MustRegister("openstack:model:loadbalancer", &LoadBalancer{})
}
