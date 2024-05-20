package models

import (
	"fmt"

	coremodels "github.com/gardener/inventory/pkg/core/models"

	"github.com/gardener/inventory/pkg/aws/constants"
)

// Region represents an AWS Region
type Region struct {
	coremodels.Base
	Name        string
	Endpoint    string
	OptInStatus string
}

// TableName implements the [gorm.io/gorm/schema.Namer] interface.
func (Region) TableName() string {
	return fmt.Sprintf("%s_%s", constants.TablePrefix, "region")
}
