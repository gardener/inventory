package models

import (
	"database/sql"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/uptrace/bun"
)

// Region represents an AWS Region
type Region struct {
	bun.BaseModel `bun:"table:aws_region"`
	coremodels.Model

	Name        string `bun:"name,notnull,unique"`
	Endpoint    string `bun:"endpoint,notnull"`
	OptInStatus string `bun:"opt_in_status,notnull"`
}

// AvailabilityZone represents an AWS Availability Zone.
type AvailabilityZone struct {
	bun.BaseModel `bun:"table:aws_az"`
	coremodels.Model

	ZoneID             string `bun:"zone_id,notnull,unique"`
	Name               string `bun:"name,notnull"`
	OptInStatus        string `bun:"opt_in_status,notnull"`
	State              string `bun:"state,notnull"`
	RegionName         string `bun:"region_name,notnull"`
	GroupName          string `bun:"group_name,notnull"`
	NetworkBorderGroup string `bun:"network_border_group,notnull"`
}

// VPC represents an AWS VPC
type VPC struct {
	bun.BaseModel `bun:"table:aws_vpc"`
	coremodels.Model

	Name       string `bun:"name,notnull"`
	VpcID      string `bun:"vpc_id,notnull,unique"`
	State      string `bun:"state,notnull"`
	IPv4CIDR   string `bun:"ipv4_cidr,notnull"`
	IPv6CIDR   string `bun:"ipv4_cidr"`
	IsDefault  bool   `bun:"is_default,notnull"`
	OwnerID    string `bun:"owner_id,notnull"`
	RegionName string `bun:"region_name,notnull"`
}

// Subnet represents an AWS Subnet
type Subnet struct {
	bun.BaseModel `bun:"table:aws_subnet"`
	coremodels.Model

	Name                   string         `bun:"name,notnull"`
	SubnetID               string         `bun:"subnet_id,notnull"`
	VpcID                  string         `bun:"vpc_id,notnull"`
	State                  string         `bun:"state,notnull"`
	AZ                     string         `bun:"az,notnull"`
	AzID                   string         `bun:"az_id,notnull"`
	AvailableIPv4Addresses int            `bun:"available_ipv4_addresses,notnull"`
	IPv4CIDR               sql.NullString `bun:"ipv4_cidr,notnull"`
	IPv6CIDR               sql.NullString `bun:"ipv6_cidr"`
}

// Instance represents an AWS EC2 instance
type Instance struct {
	bun.BaseModel `bun:"table:aws_instance"`
	coremodels.Model

	Name         string `bun:"name,notnull"`
	Arch         string `bun:"arch,notnull"`
	InstanceID   string `bun:"instance_id,notnull,unique"`
	InstanceType string `bun:"instance_type,notnull"`
	State        string `bun:"state,notnull"`
	SubnetID     string `bun:"subnet_id,notnull"`
	VpcID        string `bun:"vpc_id,notnull"`
	Platform     string `bun:"platform,notnull"`
}
