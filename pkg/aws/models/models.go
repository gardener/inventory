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

// InstanceToImage represents a link table connecting the Instance with Image
type InstanceToImage struct {
	bun.BaseModel `bun:"table:l_aws_instance_to_image"`
	coremodels.Model

	InstanceID uint64 `bun:"instance_id,notnull,unique:l_aws_instance_to_image_key"`
	ImageID    uint64 `bun:"image_id,notnull,unique:l_aws_instance_to_image_key"`
}

// ImageToRegion represents a link table connecting the Image with Region.
type ImageToRegion struct {
	bun.BaseModel `bun:"table:l_aws_image_to_region"`
	coremodels.Model

	ImageID  uint64 `bun:"image_id,notnull,unique:l_aws_image_to_region_key"`
	RegionID uint64 `bun:"region_id,notnull,unique:l_aws_image_to_region_key"`
}

// Region represents an AWS Region
type Region struct {
	bun.BaseModel `bun:"table:aws_region"`
	coremodels.Model

	Name        string `bun:"name,notnull,unique:aws_region_key"`
	AccountID   string `bun:"account_id,notnull,unique:aws_region_key"`
	Endpoint    string `bun:"endpoint,notnull"`
	OptInStatus string `bun:"opt_in_status,notnull"`
}

// AvailabilityZone represents an AWS Availability Zone.
type AvailabilityZone struct {
	bun.BaseModel `bun:"table:aws_az"`
	coremodels.Model

	ZoneID             string  `bun:"zone_id,notnull,unique:aws_az_key"`
	AccountID          string  `bun:"account_id,notnull,unique:aws_az_key"`
	ZoneType           string  `bun:"zone_type,notnull"`
	Name               string  `bun:"name,notnull"`
	OptInStatus        string  `bun:"opt_in_status,notnull"`
	State              string  `bun:"state,notnull"`
	RegionName         string  `bun:"region_name,notnull"`
	GroupName          string  `bun:"group_name,notnull"`
	NetworkBorderGroup string  `bun:"network_border_group,notnull"`
	Region             *Region `bun:"rel:has-one,join:region_name=name,join:account_id=account_id"`
}

// VPC represents an AWS VPC
type VPC struct {
	bun.BaseModel `bun:"table:aws_vpc"`
	coremodels.Model

	Name       string  `bun:"name,notnull"`
	VpcID      string  `bun:"vpc_id,notnull,unique:aws_vpc_key"`
	AccountID  string  `bun:"account_id,notnull,unique:aws_vpc_key"`
	State      string  `bun:"state,notnull"`
	IPv4CIDR   string  `bun:"ipv4_cidr,notnull"`
	IPv6CIDR   string  `bun:"ipv6_cidr,nullzero"`
	IsDefault  bool    `bun:"is_default,notnull"`
	OwnerID    string  `bun:"owner_id,notnull"`
	RegionName string  `bun:"region_name,notnull"`
	Region     *Region `bun:"rel:has-one,join:region_name=name,join:account_id=account_id"`
}

// Subnet represents an AWS Subnet
type Subnet struct {
	bun.BaseModel `bun:"table:aws_subnet"`
	coremodels.Model

	Name                   string            `bun:"name,notnull"`
	SubnetID               string            `bun:"subnet_id,notnull,unique:aws_subnet_key"`
	AccountID              string            `bun:"account_id,notnull,unique:aws_subnet_key"`
	SubnetArn              string            `bun:"subnet_arn,notnull"`
	VpcID                  string            `bun:"vpc_id,notnull"`
	State                  string            `bun:"state,notnull"`
	AZ                     string            `bun:"az,notnull"`
	AzID                   string            `bun:"az_id,notnull"`
	AvailableIPv4Addresses int               `bun:"available_ipv4_addresses,notnull"`
	IPv4CIDR               string            `bun:"ipv4_cidr,notnull"`
	IPv6CIDR               string            `bun:"ipv6_cidr,nullzero"`
	VPC                    *VPC              `bun:"rel:has-one,join:vpc_id=vpc_id,join:account_id=account_id"`
	AvailabilityZone       *AvailabilityZone `bun:"rel:has-one,join:az_id=zone_id,join:account_id=account_id"`
}

// Instance represents an AWS EC2 instance
type Instance struct {
	bun.BaseModel `bun:"table:aws_instance"`
	coremodels.Model

	Name         string    `bun:"name,notnull"`
	Arch         string    `bun:"arch,notnull"`
	InstanceID   string    `bun:"instance_id,notnull,unique:aws_instance_key"`
	AccountID    string    `bun:"account_id,notnull,unique:aws_instance_key"`
	InstanceType string    `bun:"instance_type,notnull"`
	State        string    `bun:"state,notnull"`
	SubnetID     string    `bun:"subnet_id,notnull"`
	VpcID        string    `bun:"vpc_id,notnull"`
	Platform     string    `bun:"platform,notnull"`
	RegionName   string    `bun:"region_name,notnull"`
	ImageID      string    `bun:"image_id,notnull"`
	LaunchTime   time.Time `bun:"launch_time,nullzero"`
	Region       *Region   `bun:"rel:has-one,join:region_name=name,join:account_id=account_id"`
	VPC          *VPC      `bun:"rel:has-one,join:vpc_id=vpc_id,join:account_id=account_id"`
	Subnet       *Subnet   `bun:"rel:has-one,join:subnet_id=subnet_id,join:account_id=account_id"`
	Image        *Image    `bun:"rel:has-one,join:image_id=image_id,join:account_id=account_id"`
}

// InstanceToNetworkInterface represents a link table connecting the [Instance]
// with [NetworkInterface]
type InstanceToNetworkInterface struct {
	bun.BaseModel `bun:"table:l_aws_instance_to_net_interface"`
	coremodels.Model

	InstanceID         uint64 `bun:"instance_id,notnull,unique:l_aws_instance_to_net_interface_key"`
	NetworkInterfaceID uint64 `bun:"ni_id,notnull,unique:l_aws_instance_to_net_interface_key"`
}

// Image represents an AWS AMI
type Image struct {
	bun.BaseModel `bun:"table:aws_image"`
	coremodels.Model

	ImageID        string  `bun:"image_id,notnull,unique:aws_image_key"`
	AccountID      string  `bun:"account_id,notnull,unique:aws_image_key"`
	Name           string  `bun:"name,notnull"`
	OwnerID        string  `bun:"owner_id,notnull"`
	ImageType      string  `bun:"image_type,notnull"`
	RootDeviceType string  `bun:"root_device_type,notnull"`
	Description    string  `bun:"description,notnull"`
	RegionName     string  `bun:"region_name,notnull"`
	Region         *Region `bun:"rel:has-one,join:region_name=name,join:account_id=account_id"`
}

// LoadBalancer represents an AWS load balancer
type LoadBalancer struct {
	bun.BaseModel `bun:"table:aws_loadbalancer"`
	coremodels.Model

	// ARN specifies the ARN of the Load Balancer. ARN is available only for
	// v2 Load Balancers.
	ARN string `bun:"arn,notnull"`

	// LoadBalancerID specifies the ID of the Load Balancer. The
	// LoadBalancerID is extracted from the last component of the ARN and is
	// only available for v2 LBs.
	LoadBalancerID string `bun:"load_balancer_id,notnull"`

	// State represents the state of the Load Balancer. This field is
	// present only for v2 Load Balancers.
	State string `bun:"state,notnull"`

	Name                  string  `bun:"name,notnull"`
	DNSName               string  `bun:"dns_name,notnull,unique:aws_loadbalancer_key"`
	AccountID             string  `bun:"account_id,notnull,unique:aws_loadbalancer_key"`
	CanonicalHostedZoneID string  `bun:"canonical_hosted_zone_id,notnull"`
	Scheme                string  `bun:"scheme,notnull"`
	Type                  string  `bun:"type,notnull"`
	VpcID                 string  `bun:"vpc_id,notnull"`
	VPC                   *VPC    `bun:"rel:has-one,join:vpc_id=vpc_id,join:account_id=account_id"`
	RegionName            string  `bun:"region_name,notnull"`
	Region                *Region `bun:"rel:has-one,join:region_name=name,join:account_id=account_id"`
}

// LoadBalancerToVPC represents a link table connecting the LoadBalancer with VPC.
type LoadBalancerToVPC struct {
	bun.BaseModel `bun:"table:l_aws_lb_to_vpc"`
	coremodels.Model

	LoadBalancerID uint64 `bun:"lb_id,notnull,unique:l_aws_lb_to_vpc_key"`
	VpcID          uint64 `bun:"vpc_id,notnull,unique:l_aws_lb_to_vpc_key"`
}

// LoadBalancerToRegion represents a link table connecting the LoadBalancer with Region.
type LoadBalancerToRegion struct {
	bun.BaseModel `bun:"table:l_aws_lb_to_region"`
	coremodels.Model

	LoadBalancerID uint64 `bun:"lb_id,notnull,unique:l_aws_lb_to_region_key"`
	RegionID       uint64 `bun:"region_id,notnull,unique:l_aws_lb_to_region_key"`
}

// Bucket represents an AWS S3 bucket
type Bucket struct {
	bun.BaseModel `bun:"table:aws_bucket"`
	coremodels.Model

	Name         string    `bun:"name,notnull,unique:aws_bucket_key"`
	AccountID    string    `bun:"account_id,notnull,unique:aws_bucket_key"`
	CreationDate time.Time `bun:"creation_date,notnull"`
	RegionName   string    `bun:"region_name,notnull"`
	Region       *Region   `bun:"rel:has-one,join:region_name=name,join:account_id=account_id"`
}

// NetworkInterface represents an AWS Elastic Network Interface (ENI)
type NetworkInterface struct {
	bun.BaseModel `bun:"table:aws_net_interface"`
	coremodels.Model

	Region           *Region           `bun:"rel:has-one,join:region_name=name,join:account_id=account_id"`
	RegionName       string            `bun:"region_name,notnull"`
	AZ               string            `bun:"az,notnull"`
	AvailabilityZone *AvailabilityZone `bun:"rel:has-one,join:az=name,join:account_id=account_id"`
	Description      string            `bun:"description,notnull"`
	InterfaceType    string            `bun:"interface_type,notnull"`
	MacAddress       string            `bun:"mac_address,notnull"`
	InterfaceID      string            `bun:"interface_id,notnull,unique:aws_net_interface_key"`
	AccountID        string            `bun:"account_id,notnull,unique:aws_net_interface_key"`
	OwnerID          string            `bun:"owner_id,notnull"`
	PrivateDNSName   string            `bun:"private_dns_name,notnull"`
	PrivateIPAddress string            `bun:"private_ip_address,notnull"`
	RequesterID      string            `bun:"requester_id,notnull"`
	RequesterManaged bool              `bun:"requester_managed,notnull"`
	SourceDestCheck  bool              `bun:"src_dst_check,notnull"`
	Status           string            `bun:"status,notnull"`
	Subnet           *Subnet           `bun:"rel:has-one,join:subnet_id=subnet_id,join:account_id=account_id"`
	SubnetID         string            `bun:"subnet_id,notnull"`
	VPC              *VPC              `bun:"rel:has-one,join:vpc_id=vpc_id,join:account_id=account_id"`
	VpcID            string            `bun:"vpc_id,notnull"`

	// Association
	AllocationID    string `bun:"allocation_id,notnull"`
	AssociationID   string `bun:"association_id,notnull"`
	IPOwnerID       string `bun:"ip_owner_id,notnull"`
	PublicDNSName   string `bun:"public_dns_name,notnull"`
	PublicIPAddress string `bun:"public_ip_address,notnull"`

	// Attachment
	AttachmentID        string    `bun:"attachment_id,notnull"`
	DeleteOnTermination bool      `bun:"delete_on_termination,notnull"`
	DeviceIndex         int       `bun:"device_index,notnull"`
	Instance            *Instance `bun:"rel:has-one,join:instance_id=instance_id,join:account_id=account_id"`
	InstanceID          string    `bun:"instance_id,notnull"`
	InstanceOwnerID     string    `bun:"instance_owner_id,notnull"`
	AttachmentStatus    string    `bun:"attachment_status,notnull"`
}

// LoadBalancerToNetworkInterface represents a link table connecting the
// [LoadBalancer] with [NetworkInterface].
type LoadBalancerToNetworkInterface struct {
	bun.BaseModel `bun:"table:l_aws_lb_to_net_interface"`
	coremodels.Model

	LoadBalancerID     uint64 `bun:"lb_id,notnull,unique:l_aws_lb_to_net_interface_key"`
	NetworkInterfaceID uint64 `bun:"ni_id,notnull,unique:l_aws_lb_to_net_interface_key"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("aws:model:region", &Region{})
	registry.ModelRegistry.MustRegister("aws:model:az", &AvailabilityZone{})
	registry.ModelRegistry.MustRegister("aws:model:vpc", &VPC{})
	registry.ModelRegistry.MustRegister("aws:model:subnet", &Subnet{})
	registry.ModelRegistry.MustRegister("aws:model:instance", &Instance{})
	registry.ModelRegistry.MustRegister("aws:model:image", &Image{})
	registry.ModelRegistry.MustRegister("aws:model:loadbalancer", &LoadBalancer{})
	registry.ModelRegistry.MustRegister("aws:model:bucket", &Bucket{})
	registry.ModelRegistry.MustRegister("aws:model:network_interface", &NetworkInterface{})

	// Link tables
	registry.ModelRegistry.MustRegister("aws:model:link_region_to_az", &RegionToAZ{})
	registry.ModelRegistry.MustRegister("aws:model:link_region_to_vpc", &RegionToVPC{})
	registry.ModelRegistry.MustRegister("aws:model:link_vpc_to_subnet", &VPCToSubnet{})
	registry.ModelRegistry.MustRegister("aws:model:link_vpc_to_instance", &VPCToInstance{})
	registry.ModelRegistry.MustRegister("aws:model:link_subnet_to_az", &SubnetToAZ{})
	registry.ModelRegistry.MustRegister("aws:model:link_instance_to_subnet", &InstanceToSubnet{})
	registry.ModelRegistry.MustRegister("aws:model:link_instance_to_region", &InstanceToRegion{})
	registry.ModelRegistry.MustRegister("aws:model:link_instance_to_image", &InstanceToImage{})
	registry.ModelRegistry.MustRegister("aws:model:link_image_to_region", &ImageToRegion{})
	registry.ModelRegistry.MustRegister("aws:model:link_lb_to_vpc", &LoadBalancerToVPC{})
	registry.ModelRegistry.MustRegister("aws:model:link_lb_to_region", &LoadBalancerToRegion{})
	registry.ModelRegistry.MustRegister("aws:model:link_lb_to_net_interface", &LoadBalancerToNetworkInterface{})
	registry.ModelRegistry.MustRegister("aws:model:link_instance_to_net_interface", &InstanceToNetworkInterface{})
}
