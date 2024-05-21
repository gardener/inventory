package models

import (
	"fmt"

	coremodels "github.com/gardener/inventory/pkg/core/models"

	"github.com/gardener/inventory/pkg/aws/constants"
)

// A helper function which returns a table name with prefix
func tableName(name string) string {
	return fmt.Sprintf("%s_%s", constants.TablePrefix, name)
}

// Region represents an AWS Region
type Region struct {
	coremodels.Base
	Name        string `gorm:"uniqueIndex:aws_region_name_idx"`
	Endpoint    string
	OptInStatus string
}

// TableName implements the [gorm.io/gorm/schema.Namer] interface.
func (Region) TableName() string {
	return tableName("region")
}

// AvailabilityZone represents an AWS Availability Zone.
type AvailabilityZone struct {
	coremodels.Base
	Name               string
	ZoneID             string `gorm:"uniqueIndex:aws_az_zone_id_idx"`
	OptInStatus        string
	State              string
	RegionName         string
	GroupName          string
	NetworkBorderGroup string
}

// TableName implements the [gorm.io/gorm/schema.Namer] interface.
func (AvailabilityZone) TableName() string {
	return tableName("az")
}

// VPC represents an AWS VPC
type VPC struct {
	coremodels.Base
	Name       string
	VpcID      string `gorm:"uniqueIndex:aws_vpc_vpc_id_idx"`
	State      string
	IPv4CIDR   string
	IPv6CIDR   string
	IsDefault  bool
	OwnerID    string
	RegionName string
}

// TableName implements the [gorm.io/gorm/schema.Namer] interface.
func (VPC) TableName() string {
	return tableName("vpc")
}
