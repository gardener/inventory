// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"strings"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"

	"github.com/gardener/inventory/pkg/azure/constants"
	"github.com/gardener/inventory/pkg/azure/models"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// GetResourceGroupsFromDB returns the [models.ResourceGroup] from the database.
func GetResourceGroupsFromDB(ctx context.Context) ([]models.ResourceGroup, error) {
	items := make([]models.ResourceGroup, 0)
	err := db.DB.NewSelect().Model(&items).Scan(ctx)
	return items, err
}

// GetVPCsFromDB returns the [models.VPC] from the database.
func GetVPCsFromDB(ctx context.Context) ([]models.VPC, error) {
	items := make([]models.VPC, 0)
	err := db.DB.NewSelect().Model(&items).Scan(ctx)
	return items, err
}

// GetPowerState returns the power state of a Virtual Machine by looking up the
// provided states.
func GetPowerState(states []*armcompute.InstanceViewStatus) string {
	if states == nil {
		return constants.PowerStateUnknown
	}

	powerStatePrefix := "PowerState/"
	for _, state := range states {
		code := ptr.Value(state.Code, "")
		if strings.HasPrefix(code, powerStatePrefix) {
			return strings.TrimPrefix(code, powerStatePrefix)
		}
	}

	return constants.PowerStateUnknown
}
