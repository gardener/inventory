package models

import (
	"fmt"

	coremodels "github.com/gardener/inventory/pkg/core/models"

	"github.com/gardener/inventory/pkg/aws/constants"
)

// Region represents an AWS Region
type Region struct {
	coremodels.Base
	Name        string `gorm:"uniqueIndex:aws_region_name_idx"`
	Endpoint    string
	OptInStatus string
}

// TableName implements the [gorm.io/gorm/schema.Namer] interface.
func (Region) TableName() string {
	return fmt.Sprintf("%s_%s", constants.TablePrefix, "region")
}
