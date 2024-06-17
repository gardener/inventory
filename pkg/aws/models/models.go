// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// RegionToAZ represents a link table connecting the Region with AZ.
type RegionToAZ struct {
	bun.BaseModel `bun:"table:l_aws_region_to_az"`
	coremodels.Model

	RegionID           uint64 `bun:"region_id,notnull,unique:l_aws_region_to_az_key"`
	AvailabilityZoneID uint64 `bun:"az_id,notnull,unique:l_aws_region_to_az_key"`
}

// RegionToVPC represents a link table connecting the Region with VPC.
type RegionToVPC struct {
	bun.BaseModel `bun:"table:l_aws_region_to_vpc"`
	coremodels.Model

	RegionID uint64 `bun:"region_id,notnull,unique:l_aws_region_to_vpc_key"`
	VpcID    uint64 `bun:"vpc_id,notnull,unique:l_aws_region_to_vpc_key"`
}

// VPCToSubnet represents a link table connecting the VPC with Subnet.
type VPCToSubnet struct {
	bun.BaseModel `bun:"table:l_aws_vpc_to_subnet"`
	coremodels.Model

	VpcID    uint64 `bun:"vpc_id,notnull,unique:l_aws_vpc_to_subnet_key"`
	SubnetID uint64 `bun:"subnet_id,notnull,unique:l_aws_vpc_to_subnet_key"`
}

// VPCToInstance represents a link table connecting the VPC with Instance.
type VPCToInstance struct {
	bun.BaseModel `bun:"table:l_aws_vpc_to_instance"`
	coremodels.Model

	VpcID      uint64 `bun:"vpc_id,notnull,unique:l_aws_vpc_to_instance_key"`
	InstanceID uint64 `bun:"instance_id,notnull,unique:l_aws_vpc_to_instance_key"`
}

// SubnetToAZ represents a link table connecting the Subnet with AZ.
type SubnetToAZ struct {
	bun.BaseModel `bun:"table:l_aws_subnet_to_az"`
	coremodels.Model

	AvailabilityZoneID uint64 `bun:"az_id,notnull,unique:l_aws_subnet_to_az_key"`
	SubnetID           uint64 `bun:"subnet_id,notnull,unique:l_aws_subnet_to_az_key"`
}

// InstanceToSubnet represents a link table connecting the Instance with Subnet.
type InstanceToSubnet struct {
	bun.BaseModel `bun:"table:l_aws_instance_to_subnet"`
	coremodels.Model

	InstanceID uint64 `bun:"instance_id,notnull,unique:l_aws_instance_to_subnet_key"`
	SubnetID   uint64 `bun:"subnet_id,notnull,unique:l_aws_instance_to_subnet_key"`
}

// InstanceToRegion represents a link table connecting the Instance with Region.
type InstanceToRegion struct {
	bun.BaseModel `bun:"table:l_aws_instance_to_region"`
	coremodels.Model

	InstanceID uint64 `bun:"instance_id,notnull,unique:l_aws_instance_to_region_key"`
	RegionID   uint64 `bun:"region_id,notnull,unique:l_aws_instance_to_region_key"`
}

// Region represents an AWS Region
type Region struct {
	bun.BaseModel `bun:"table:aws_region"`
	coremodels.Model

	Name        string              `bun:"name,notnull,unique"`
	Endpoint    string              `bun:"endpoint,notnull"`
	OptInStatus string              `bun:"opt_in_status,notnull"`
	Zones       []*AvailabilityZone `bun:"rel:has-many,join:name=region_name"`
	VPCs        []*VPC              `bun:"rel:has-many,join:name=region_name"`
	Instances   []*Instance         `bun:"rel:has-many,join:name=region_name"`
}

// AvailabilityZone represents an AWS Availability Zone.
type AvailabilityZone struct {
	bun.BaseModel `bun:"table:aws_az"`
	coremodels.Model

	ZoneID             string  `bun:"zone_id,notnull,unique"`
	ZoneType           string  `bun:"zone_type,notnull"`
	Name               string  `bun:"name,notnull"`
	OptInStatus        string  `bun:"opt_in_status,notnull"`
	State              string  `bun:"state,notnull"`
	RegionName         string  `bun:"region_name,notnull"`
	GroupName          string  `bun:"group_name,notnull"`
	NetworkBorderGroup string  `bun:"network_border_group,notnull"`
	Region             *Region `bun:"rel:has-one,join:region_name=name"`
}

// VPC represents an AWS VPC
type VPC struct {
	bun.BaseModel `bun:"table:aws_vpc"`
	coremodels.Model

	Name       string      `bun:"name,notnull"`
	VpcID      string      `bun:"vpc_id,notnull,unique"`
	State      string      `bun:"state,notnull"`
	IPv4CIDR   string      `bun:"ipv4_cidr,notnull"`
	IPv6CIDR   string      `bun:"ipv6_cidr,nullzero"`
	IsDefault  bool        `bun:"is_default,notnull"`
	OwnerID    string      `bun:"owner_id,notnull"`
	RegionName string      `bun:"region_name,notnull"`
	Region     *Region     `bun:"rel:has-one,join:region_name=name"`
	Subnets    []*Subnet   `bun:"rel:has-many,join:vpc_id=vpc_id"`
	Instances  []*Instance `bun:"rel:has-many,join:vpc_id=vpc_id"`
}

// Subnet represents an AWS Subnet
type Subnet struct {
	bun.BaseModel `bun:"table:aws_subnet"`
	coremodels.Model

	Name                   string            `bun:"name,notnull"`
	SubnetID               string            `bun:"subnet_id,notnull,unique"`
	SubnetArn              string            `bun:"subnet_arn,notnull"`
	VpcID                  string            `bun:"vpc_id,notnull"`
	State                  string            `bun:"state,notnull"`
	AZ                     string            `bun:"az,notnull"`
	AzID                   string            `bun:"az_id,notnull"`
	AvailableIPv4Addresses int               `bun:"available_ipv4_addresses,notnull"`
	IPv4CIDR               string            `bun:"ipv4_cidr,notnull"`
	IPv6CIDR               string            `bun:"ipv6_cidr,nullzero"`
	VPC                    *VPC              `bun:"rel:has-one,join:vpc_id=vpc_id"`
	AvailabilityZone       *AvailabilityZone `bun:"rel:has-one,join:az_id=zone_id"`
	Instances              []*Instance       `bun:"rel:has-many,join:subnet_id=subnet_id"`
}

// Instance represents an AWS EC2 instance
type Instance struct {
	bun.BaseModel `bun:"table:aws_instance"`
	coremodels.Model

	Name         string  `bun:"name,notnull"`
	Arch         string  `bun:"arch,notnull"`
	InstanceID   string  `bun:"instance_id,notnull,unique"`
	InstanceType string  `bun:"instance_type,notnull"`
	State        string  `bun:"state,notnull"`
	SubnetID     string  `bun:"subnet_id,notnull"`
	VpcID        string  `bun:"vpc_id,notnull"`
	Platform     string  `bun:"platform,notnull"`
	RegionName   string  `bun:"region_name,notnull"`
	Region       *Region `bun:"rel:has-one,join:region_name=name"`
	VPC          *VPC    `bun:"rel:has-one,join:vpc_id=vpc_id"`
	Subnet       *Subnet `bun:"rel:has-one,join:subnet_id=subnet_id"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("aws:model:region", &Region{})
	registry.ModelRegistry.MustRegister("aws:model:az", &AvailabilityZone{})
	registry.ModelRegistry.MustRegister("aws:model:vpc", &VPC{})
	registry.ModelRegistry.MustRegister("aws:model:subnet", &Subnet{})
	registry.ModelRegistry.MustRegister("aws:model:instance", &Instance{})

	// Link tables
	registry.ModelRegistry.MustRegister("aws:model:link_region_to_az", &RegionToAZ{})
	registry.ModelRegistry.MustRegister("aws:model:link_region_to_vpc", &RegionToVPC{})
	registry.ModelRegistry.MustRegister("aws:model:link_vpc_to_subnet", &VPCToSubnet{})
	registry.ModelRegistry.MustRegister("aws:model:link_vpc_to_instance", &VPCToInstance{})
	registry.ModelRegistry.MustRegister("aws:model:link_subnet_to_az", &SubnetToAZ{})
	registry.ModelRegistry.MustRegister("aws:model:link_instance_to_subnet", &InstanceToSubnet{})
	registry.ModelRegistry.MustRegister("aws:model:link_instance_to_region", &InstanceToRegion{})
}
