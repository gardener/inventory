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

// Names for the models provided by this package,
// used for registering models with [registry.ModelRegistry]
const (
	ServerModelName               = "openstack:model:server"
	NetworkModelName              = "openstack:model:network"
	LoadBalancerModelName         = "openstack:model:loadbalancer"
	LoadBalancerWithPoolModelName = "openstack:model:loadbalancer_with_pool"
	SubnetModelName               = "openstack:model:subnet"
	FloatingIPModelName           = "openstack:model:floating_ip"
	ProjectModelName              = "openstack:model:project"
	PortModelName                 = "openstack:model:port"
	PortIPModelName               = "openstack:model:port_ip"
	RouterModelName               = "openstack:model:router"
	RouterExternalIPModelName     = "openstack:model:router_external_ip"
	PoolModelName                 = "openstack:model:pool"
	PoolMemberModelName           = "openstack:model:pool_member"
	ContainerModelName            = "openstack:model:container"
	ObjectModelName               = "openstack:model:object"
	VolumeModelName               = "openstack:model:volume"
	VolumeAttachmentModelName     = "openstack:model:volume_attachment"

	SubnetToNetworkModelName       = "openstack:model:link_subnet_to_network"
	SubnetToProjectModelName       = "openstack:model:link_subnet_to_project"
	ServerToProjectModelName       = "openstack:model:link_server_to_project"
	ServerToNetworkModelName       = "openstack:model:link_server_to_network"
	LoadBalancerToSubnetModelName  = "openstack:model:link_loadbalancer_to_subnet"
	LoadBalancerToNetworkModelName = "openstack:model:link_loadbalancer_to_network"
	LoadBalancerToProjectModelName = "openstack:model:link_loadbalancer_to_project"
	NetworkToProjectModelName      = "openstack:model:link_network_to_project"
	PortToServerModelName          = "openstack:model:link_server_to_port"
)

// models specifies the mapping between name and model type, which will be
// registered with [registry.ModelRegistry].
var models = map[string]any{
	ServerModelName:               &Server{},
	NetworkModelName:              &Network{},
	LoadBalancerModelName:         &LoadBalancer{},
	LoadBalancerWithPoolModelName: &LoadBalancerWithPool{},
	SubnetModelName:               &Subnet{},
	FloatingIPModelName:           &FloatingIP{},
	ProjectModelName:              &Project{},
	PortModelName:                 &Port{},
	PortIPModelName:               &PortIP{},
	RouterModelName:               &Router{},
	RouterExternalIPModelName:     &RouterExternalIP{},
	PoolModelName:                 &Pool{},
	PoolMemberModelName:           &PoolMember{},
	ContainerModelName:            &Container{},
	ObjectModelName:               &Object{},
	VolumeModelName:               &Volume{},
	VolumeAttachmentModelName:     &VolumeAttachment{},

	// Link models
	SubnetToNetworkModelName:       &SubnetToNetwork{},
	SubnetToProjectModelName:       &SubnetToProject{},
	ServerToProjectModelName:       &ServerToProject{},
	ServerToNetworkModelName:       &ServerToNetwork{},
	LoadBalancerToSubnetModelName:  &LoadBalancerToSubnet{},
	LoadBalancerToNetworkModelName: &LoadBalancerToNetwork{},
	LoadBalancerToProjectModelName: &LoadBalancerToProject{},
	NetworkToProjectModelName:      &NetworkToProject{},
	PortToServerModelName:          &PortToServer{},
}

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
	Project           *Project  `bun:"rel:has-one,join:project_id=project_id"`
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

// PortToServer represents a link table connecting Ports with Servers.
type PortToServer struct {
	bun.BaseModel `bun:"table:l_openstack_port_to_server"`
	coremodels.Model

	PortID   uuid.UUID `bun:"port_id,notnull"`
	ServerID uuid.UUID `bun:"server_id,notnull"`
}

// ServerToNetwork represents a link table connecting Servers with Networks.
type ServerToNetwork struct {
	bun.BaseModel `bun:"table:l_openstack_server_to_network"`
	coremodels.Model

	ServerID  uuid.UUID `bun:"server_id,notnull"`
	NetworkID uuid.UUID `bun:"network_id,notnull"`
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

// Port represents an OpenStack Port.
type Port struct {
	bun.BaseModel `bun:"table:openstack_port"`
	coremodels.Model

	PortID      string    `bun:"port_id,notnull,unique:openstack_port_key"`
	Name        string    `bun:"name,notnull"`
	ProjectID   string    `bun:"project_id,notnull,unique:openstack_port_key"`
	NetworkID   string    `bun:"network_id,notnull,unique:openstack_port_key"`
	DeviceID    string    `bun:"device_id,notnull"`
	DeviceOwner string    `bun:"device_owner,notnull"`
	Domain      string    `bun:"domain,notnull"`
	Region      string    `bun:"region,notnull,unique:openstack_port_key"`
	MacAddress  string    `bun:"mac_address,notnull"`
	Status      string    `bun:"status,notnull"`
	Description string    `bun:"description,notnull"`
	TimeCreated time.Time `bun:"port_created_at,notnull"`
	TimeUpdated time.Time `bun:"port_updated_at,notnull"`
	Network     *Network  `bun:"rel:has-one,join:network_id=network_id,join:project_id=project_id"`
	Project     *Project  `bun:"rel:has-one,join:project_id=project_id"`
	Server      *Server   `bun:"rel:has-one,join:device_id=server_id,join:project_id=project_id"`
}

// PortIP represents an OpenStack Port IP address.
type PortIP struct {
	bun.BaseModel `bun:"table:openstack_port_ip"`
	coremodels.Model

	PortID    string  `bun:"port_id,notnull,unique:openstack_port_ip_key"`
	ProjectID string  `bun:"project_id,notnull,unique:openstack_port_ip_key"`
	IPAddress net.IP  `bun:"ip_address,nullzero,type:inet,unique:openstack_port_ip_key"`
	SubnetID  string  `bun:"subnet_id,notnull,unique:openstack_port_ip_key"`
	Port      *Port   `bun:"rel:has-one,join:port_id=port_id,join:project_id=project_id"`
	Subnet    *Subnet `bun:"rel:has-one,join:subnet_id=subnet_id,join:project_id=project_id"`
}

// Router represents an OpenStack Router.
type Router struct {
	bun.BaseModel `bun:"table:openstack_router"`
	coremodels.Model

	RouterID          string   `bun:"router_id,notnull,unique:openstack_router_key"`
	Name              string   `bun:"name,notnull"`
	ProjectID         string   `bun:"project_id,notnull,unique:openstack_router_key"`
	Domain            string   `bun:"domain,notnull"`
	Region            string   `bun:"region,notnull"`
	Status            string   `bun:"status,notnull"`
	Description       string   `bun:"description,notnull"`
	ExternalNetworkID string   `bun:"external_network_id,notnull"`
	Project           *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// RouterExternalIP represents an external IP for a OpenStack router.
type RouterExternalIP struct {
	bun.BaseModel `bun:"table:openstack_router_external_ip"`
	coremodels.Model

	RouterID         string   `bun:"router_id,notnull,unique:openstack_router_external_ip_key"`
	ProjectID        string   `bun:"project_id,notnull,unique:openstack_router_external_ip_key"`
	ExternalIP       net.IP   `bun:"external_ip,nullzero,type:inet,unique:openstack_router_external_ip_key"`
	ExternalSubnetID string   `bun:"external_subnet_id,notnull,unique:openstack_router_external_ip_key"`
	Project          *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// Container represents an OpenStack Container.
type Container struct {
	bun.BaseModel `bun:"table:openstack_container"`
	coremodels.Model

	Name        string `bun:"name,notnull,unique:openstack_container_key"`
	ProjectID   string `bun:"project_id,notnull,unique:openstack_container_key"`
	Bytes       int64  `bun:"bytes,notnull"`
	ObjectCount int64  `bun:"object_count,notnull"`
}

// Object represents an OpenStack Object.
type Object struct {
	bun.BaseModel `bun:"table:openstack_object"`
	coremodels.Model

	Name          string    `bun:"name,notnull,unique:openstack_object_key"`
	ProjectID     string    `bun:"project_id,notnull,unique:openstack_object_key"`
	ContainerName string    `bun:"container_name,notnull,unique:openstack_object_key"`
	ContentType   string    `bun:"content_type,notnull"`
	LastModified  time.Time `bun:"last_modified,notnull"`
	IsLatest      bool      `bun:"is_latest,notnull"`
}

// Pool represents an OpenStack server Pool.
type Pool struct {
	bun.BaseModel `bun:"table:openstack_pool"`
	coremodels.Model

	PoolID      string `bun:"pool_id,notnull,unique:openstack_pool_key"`
	ProjectID   string `bun:"project_id,notnull,unique:openstack_pool_key"`
	Name        string `bun:"name,notnull"`
	SubnetID    string `bun:"subnet_id,notnull"`
	Description string `bun:"description,notnull"`
}

// PoolMember represents device that is a member of an OpenStack pool.
type PoolMember struct {
	bun.BaseModel `bun:"table:openstack_pool_member"`
	coremodels.Model

	MemberID              string    `bun:"member_id,notnull,unique:openstack_pool_member_key"`
	PoolID                string    `bun:"pool_id,notnull,unique:openstack_pool_member_key"`
	ProjectID             string    `bun:"project_id,notnull,unique:openstack_pool_member_key"`
	Name                  string    `bun:"name,notnull"`
	InferredGardenerShoot string    `bun:"inferred_gardener_shoot,nullzero"`
	SubnetID              string    `bun:"subnet_id,notnull"`
	ProtocolPort          int       `bun:"protocol_port,notnull"`
	MemberCreatedAt       time.Time `bun:"member_created_at,notnull"`
	MemberUpdatedAt       time.Time `bun:"member_updated_at,notnull"`
	Pool                  *Pool     `bun:"rel:has-one,join:project_id=project_id,join:pool_id=pool_id"`
}

// LoadBalancerWithPool represents the connection between an OpenStack LoadBalancer and Pool
// This is different to a link table in that it is populated during collection
// and the pool ID is not a forein key in the Pool table, as the Pool record
// might not exist yet.
type LoadBalancerWithPool struct {
	bun.BaseModel `bun:"table:openstack_loadbalancer_with_pool"`
	coremodels.Model

	LoadBalancerID string        `bun:"loadbalancer_id,notnull,unique:openstack_loadbalancer_with_pool_key"`
	PoolID         string        `bun:"pool_id,notnull,unique:openstack_loadbalancer_with_pool_key"`
	ProjectID      string        `bun:"project_id,notnull,unique:openstack_loadbalancer_with_pool_key"`
	LoadBalancer   *LoadBalancer `bun:"rel:has-one,join:project_id=project_id,join:loadbalancer_id=loadbalancer_id"`
	Pool           *Pool         `bun:"rel:has-one,join:project_id=project_id,join:pool_id=pool_id"`
}

// Volume represents an OpenStack Volume.
type Volume struct {
	bun.BaseModel `bun:"table:openstack_volume"`
	coremodels.Model

	VolumeID          string    `bun:"volume_id,notnull,unique:openstack_volume_key"`
	Name              string    `bun:"name,notnull"`
	ProjectID         string    `bun:"project_id,notnull,unique:openstack_volume_key"`
	Domain            string    `bun:"domain,notnull,unique:openstack_volume_key"`
	Region            string    `bun:"region,notnull,unique:openstack_volume_key"`
	UserID            string    `bun:"user_id,notnull"`
	AvailabilityZone  string    `bun:"availability_zone,notnull"`
	Size              int       `bun:"size,notnull"`
	VolumeType        string    `bun:"volume_type,notnull"`
	Status            string    `bun:"status,notnull"`
	ReplicationStatus string    `bun:"replication_status,notnull"`
	Bootable          string    `bun:"bootable,notnull"`
	Encrypted         bool      `bun:"encrypted,notnull"`
	MultiAttach       bool      `bun:"multi_attach,notnull"`
	SnapshotID        string    `bun:"snapshot_id,notnull"`
	Description       string    `bun:"description,notnull"`
	TimeCreated       time.Time `bun:"volume_created_at,notnull"`
	TimeUpdated       time.Time `bun:"volume_updated_at,notnull"`
}

// VolumeAttachment represents an OpenStack Volume attachment.
// A single volume can have multiple attachments.
type VolumeAttachment struct {
	bun.BaseModel `bun:"table:openstack_volume_attachment"`
	coremodels.Model

	AttachmentID string    `bun:"attachment_id,notnull,unique:openstack_volume_attachment_key"`
	VolumeID     string    `bun:"volume_id,notnull"`
	AttachedAt   time.Time `bun:"attached_at,notnull"`
	Device       string    `bun:"device,notnull"`
	Hostname     string    `bun:"hostname,notnull"`
	ServerID     string    `bun:"server_id,notnull"`
}

func init() {
	// Register the models with the default registry

	for k, v := range models {
		registry.ModelRegistry.MustRegister(k, v)
	}
}
