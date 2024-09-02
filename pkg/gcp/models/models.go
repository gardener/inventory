// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"net"
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
	// "projects/<uint64>" value
	Name string `bun:"name,notnull,unique"`
	// ProjectID is the user-defined globally unique project id.
	ProjectID string `bun:"project_id,notnull,unique"`

	Parent            string      `bun:"parent,notnull"`
	State             string      `bun:"state,notnull"`
	DisplayName       string      `bun:"display_name,notnull"`
	ProjectCreateTime time.Time   `bun:"project_create_time,nullzero"`
	ProjectUpdateTime time.Time   `bun:"project_update_time,nullzero"`
	ProjectDeleteTime time.Time   `bun:"project_delete_time,nullzero"`
	Etag              string      `bun:"etag,notnull"`
	Instances         []*Instance `bun:"rel:has-many,join:project_id=project_id"`
	VPCs              []*VPC      `bun:"rel:has-many,join:project_id=project_id"`
}

// Instance represents a GCP Instance.
type Instance struct {
	bun.BaseModel `bun:"table:gcp_instance"`
	coremodels.Model

	Name                 string   `bun:"name,notnull"`
	Hostname             string   `bun:"hostname,notnull"`
	InstanceID           uint64   `bun:"instance_id,notnull,unique:gcp_instance_key"`
	ProjectID            string   `bun:"project_id,notnull,unique:gcp_instance_key"`
	Zone                 string   `bun:"zone,notnull"`
	Region               string   `bun:"region,notnull"`
	CanIPForward         bool     `bun:"can_ip_forward,notnull"`
	CPUPlatform          string   `bun:"cpu_platform,notnull"`
	CreationTimestamp    string   `bun:"creation_timestamp,nullzero"`
	Description          string   `bun:"description,notnull"`
	LastStartTimestamp   string   `bun:"last_start_timestamp,nullzero"`
	LastStopTimestamp    string   `bun:"last_stop_timestamp,nullzero"`
	LastSuspendTimestamp string   `bun:"last_suspend_timestamp,nullzero"`
	MachineType          string   `bun:"machine_type,notnull"`
	MinCPUPlatform       string   `bun:"min_cpu_platform,notnull"`
	SelfLink             string   `bun:"self_link,notnull"`
	SourceMachineImage   string   `bun:"source_machine_image,notnull"`
	Status               string   `bun:"status,notnull"`
	StatusMessage        string   `bun:"status_message,notnull"`
	Project              *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// InstanceToProject represents a link table connecting the [Project] with
// [Instance] models.
type InstanceToProject struct {
	bun.BaseModel `bun:"table:l_gcp_instance_to_project"`
	coremodels.Model

	ProjectID  uint64 `bun:"project_id,notnull,unique:l_gcp_instance_to_project_key"`
	InstanceID uint64 `bun:"instance_id,notnull,unique:l_gcp_instance_to_project_key"`
}

// VPC represents a GCP VPC
type VPC struct {
	bun.BaseModel `bun:"table:gcp_vpc"`
	coremodels.Model

	VPCID             uint64   `bun:"vpc_id,notnull,unique:gcp_vpc_key"`
	ProjectID         string   `bun:"project_id,notnull,unique:gcp_vpc_key"`
	Name              string   `bun:"name,notnull"`
	CreationTimestamp string   `bun:"creation_timestamp,nullzero"`
	Description       string   `bun:"description,notnull"`
	GatewayIPv4       string   `bun:"gateway_ipv4,notnull"`
	FirewallPolicy    string   `bun:"firewall_policy,notnull"`
	MTU               int32    `bun:"mtu,notnull"`
	Project           *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// VPCToProject represents a link table connecting the [Project] with
// [VPC] models.
type VPCToProject struct {
	bun.BaseModel `bun:"table:l_gcp_vpc_to_project"`
	coremodels.Model

	ProjectID uint64 `bun:"project_id,notnull,unique:l_gcp_vpc_to_project_key"`
	VPCID     uint64 `bun:"vpc_id,notnull,unique:l_gcp_vpc_to_project_key"`
}

// Address represents a GCP static IP address resource. The Address model
// represents both - a global (external and internal) and regional (external and
// internal) IP address.
type Address struct {
	bun.BaseModel `bun:"table:gcp_address"`
	coremodels.Model

	Address           net.IP   `bun:"address,notnull,type:varchar"`
	AddressType       string   `bun:"address_type,notnull"`
	IsGlobal          bool     `bun:"is_global,notnull"`
	CreationTimestamp string   `bun:"creation_timestamp,nullzero"`
	Description       string   `bun:"description,notnull"`
	AddressID         uint64   `bun:"address_id,notnull,unique:gcp_address_key"`
	ProjectID         string   `bun:"project_id,notnull,unique:gcp_address_key"`
	Region            string   `bun:"region,notnull"`
	IPVersion         string   `bun:"ip_version,notnull"`
	IPv6EndpointType  string   `bun:"ipv6_endpoint_type,notnull"`
	Name              string   `bun:"name,notnull"`
	Network           string   `bun:"network,notnull"`
	NetworkTier       string   `bun:"network_tier,notnull"`
	Subnetwork        string   `bun:"subnetwork,notnull"`
	PrefixLength      int      `bun:"prefix_length,notnull"`
	Purpose           string   `bun:"purpose,notnull"`
	SelfLink          string   `bun:"self_link,notnull"`
	Status            string   `bun:"status,notnull"`
	Project           *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// AddressToProject represents a link table connecting the [Project] with
// [Address] models.
type AddressToProject struct {
	bun.BaseModel `bun:"table:l_gcp_addr_to_project"`
	coremodels.Model

	ProjectID uint64 `bun:"project_id,notnull,unique:l_gcp_addr_to_project_key"`
	AddressID uint64 `bun:"address_id,notnull,unique:l_gcp_addr_to_project_key"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("gcp:model:project", &Project{})
	registry.ModelRegistry.MustRegister("gcp:model:instance", &Instance{})
	registry.ModelRegistry.MustRegister("gcp:model:vpc", &VPC{})
	registry.ModelRegistry.MustRegister("gcp:model:address", &Address{})

	// Link tables
	registry.ModelRegistry.MustRegister("gcp:model:link_instance_to_project", &InstanceToProject{})
	registry.ModelRegistry.MustRegister("gcp:model:link_vpc_to_project", &VPCToProject{})
	registry.ModelRegistry.MustRegister("gcp:model:link_addr_to_project", &AddressToProject{})
}
