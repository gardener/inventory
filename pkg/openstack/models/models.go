// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"net"
	"time"

	"github.com/google/uuid"
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
	Project          *Project  `bun:"rel:has-one,join:project_id=project_id"`
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
	Subnets     []*Subnet `bun:"rel:has-many,join:network_id=network_id,join:project_id=project_id"`
	Project     *Project  `bun:"rel:has-one,join:project_id=project_id"`
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
	Subnet         *Subnet   `bun:"rel:has-one,join:vip_subnet_id=subnet_id,join:project_id=project_id"`
	Project        *Project  `bun:"rel:has-one,join:project_id=project_id"`
	Network        *Network  `bun:"rel:has-one,join:vip_network_id=network_id,join:project_id=project_id"`
}

// Subnet represents an OpenStack Subnet.
type Subnet struct {
	bun.BaseModel `bun:"table:openstack_subnet"`
	coremodels.Model

	SubnetID     string   `bun:"subnet_id,notnull,unique:openstack_subnet_key"`
	Name         string   `bun:"name,notnull"`
	ProjectID    string   `bun:"project_id,notnull,unique:openstack_subnet_key"`
	Domain       string   `bun:"domain,notnull"`
	Region       string   `bun:"region,notnull"`
	NetworkID    string   `bun:"network_id,notnull"`
	GatewayIP    string   `bun:"gateway_ip,notnull"`
	CIDR         string   `bun:"cidr,notnull"`
	SubnetPoolID string   `bun:"subnet_pool_id,notnull"`
	EnableDHCP   bool     `bun:"enable_dhcp,notnull"`
	IPVersion    int      `bun:"ip_version,notnull"`
	Description  string   `bun:"description,notnull"`
	Network      *Network `bun:"rel:has-one,join:network_id=network_id,join:project_id=project_id"`
	Project      *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// FloatingIP represents an OpenStack Floating IP.
type FloatingIP struct {
	bun.BaseModel `bun:"table:openstack_floating_ip"`
	coremodels.Model

	FloatingIPID      string    `bun:"floating_ip_id,notnull,unique:openstack_floating_ip_key"`
	ProjectID         string    `bun:"project_id,notnull,unique:openstack_floating_ip_key"`
	Domain            string    `bun:"domain,notnull"`
	Region            string    `bun:"region,notnull"`
	FloatingIP        net.IP    `bun:"floating_ip,notnull"`
	FloatingNetworkID string    `bun:"floating_network_id,notnull"`
	PortID            string    `bun:"port_id,notnull"`
	RouterID          string    `bun:"router_id,notnull"`
	FixedIP           net.IP    `bun:"fixed_ip,notnull"`
	Description       string    `bun:"description,notnull"`
	TimeCreated       time.Time `bun:"ip_created_at,notnull"`
	TimeUpdated       time.Time `bun:"ip_updated_at,notnull"`
}

// SubnetToNetwork represents a link table connecting Subnets with Networks.
type SubnetToNetwork struct {
	bun.BaseModel `bun:"table:l_openstack_subnet_to_network"`
	coremodels.Model

	SubnetID  uuid.UUID `bun:"subnet_id,notnull"`
	NetworkID uuid.UUID `bun:"network_id,notnull"`
}

// SubnetToProject represents a link table connecting Subnets with Projects.
type SubnetToProject struct {
	bun.BaseModel `bun:"table:l_openstack_subnet_to_project"`
	coremodels.Model

	SubnetID  uuid.UUID `bun:"subnet_id,notnull"`
	ProjectID uuid.UUID `bun:"project_id,notnull"`
}

// LoadBalancerToSubnet represents a link table connecting LoadBalancers with Subnets.
type LoadBalancerToSubnet struct {
	bun.BaseModel `bun:"table:l_openstack_loadbalancer_to_subnet"`
	coremodels.Model

	LoadBalancerID uuid.UUID `bun:"lb_id,notnull"`
	SubnetID       uuid.UUID `bun:"subnet_id,notnull"`
}

// LoadBalancerToProject represents a link table connecting LoadBalancers with Projects.
type LoadBalancerToProject struct {
	bun.BaseModel `bun:"table:l_openstack_loadbalancer_to_project"`
	coremodels.Model

	LoadBalancerID uuid.UUID `bun:"lb_id,notnull"`
	ProjectID      uuid.UUID `bun:"project_id,notnull"`
}

// LoadBalancerToNetwork represents a link table connecting LoadBalancers with Networks.
type LoadBalancerToNetwork struct {
	bun.BaseModel `bun:"table:l_openstack_loadbalancer_to_network"`
	coremodels.Model

	LoadBalancerID uuid.UUID `bun:"lb_id,notnull"`
	NetworkID      uuid.UUID `bun:"network_id,notnull"`
}

// ServerToProject represents a link table connecting Servers with Projects.
type ServerToProject struct {
	bun.BaseModel `bun:"table:l_openstack_server_to_project"`
	coremodels.Model

	ServerID  uuid.UUID `bun:"server_id,notnull"`
	ProjectID uuid.UUID `bun:"project_id,notnull"`
}

// NetworkToProject represents a link table connecting Networks with Projects.
type NetworkToProject struct {
	bun.BaseModel `bun:"table:l_openstack_network_to_project"`
	coremodels.Model

	NetworkID uuid.UUID `bun:"network_id,notnull"`
	ProjectID uuid.UUID `bun:"project_id,notnull"`
}

// Project represents an OpenStack Project.
type Project struct {
	bun.BaseModel `bun:"table:openstack_project"`
	coremodels.Model

	ProjectID   string `bun:"project_id,notnull,unique:openstack_project_key"`
	Name        string `bun:"name,notnull"`
	Domain      string `bun:"domain,notnull"`
	Region      string `bun:"region,notnull"`
	ParentID    string `bun:"parent_id,notnull"`
	Description string `bun:"description,notnull"`
	Enabled     bool   `bun:"enabled,notnull"`
	IsDomain    bool   `bun:"is_domain,notnull"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("openstack:model:server", &Server{})
	registry.ModelRegistry.MustRegister("openstack:model:network", &Network{})
	registry.ModelRegistry.MustRegister("openstack:model:loadbalancer", &LoadBalancer{})
	registry.ModelRegistry.MustRegister("openstack:model:subnet", &Subnet{})
	registry.ModelRegistry.MustRegister("openstack:model:floating_ip", &FloatingIP{})
	registry.ModelRegistry.MustRegister("openstack:model:project", &Project{})
	registry.ModelRegistry.MustRegister("openstack:model:link_subnet_to_network", &SubnetToNetwork{})
	registry.ModelRegistry.MustRegister("openstack:model:link_loadbalancer_to_subnet", &LoadBalancerToSubnet{})
	registry.ModelRegistry.MustRegister("openstack:model:link_server_to_project", &ServerToProject{})
	registry.ModelRegistry.MustRegister("openstack:model:link_loadbalancer_to_project", &LoadBalancerToProject{})
	registry.ModelRegistry.MustRegister("openstack:model:link_loadbalancer_to_network", &LoadBalancerToNetwork{})
	registry.ModelRegistry.MustRegister("openstack:model:link_network_to_project", &NetworkToProject{})
	registry.ModelRegistry.MustRegister("openstack:model:link_subnet_to_project", &SubnetToProject{})
}
