// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/hibiken/asynq"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"

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

// GetStorageAccountsFromDB returns the [models.StorageAccount] from the database.
func GetStorageAccountsFromDB(ctx context.Context) ([]models.StorageAccount, error) {
	items := make([]models.StorageAccount, 0)
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

// MaybeSkipRetry wraps known "good" Azure errors with [asynq.SkipRetry], so
// that the tasks from which these errors originate from won't be retried.
func MaybeSkipRetry(err error) error {
	// Skip retrying for the following HTTP status codes
	skipRetryCodes := []int{
		http.StatusNotFound,
		http.StatusBadRequest,
	}

	// Azure SDK errors
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		if slices.Contains(skipRetryCodes, respErr.StatusCode) {
			return fmt.Errorf("%w (%w)", err, asynq.SkipRetry)
		}
	}

	// Graph SDK errors
	var graphRespErr *odataerrors.ODataError
	if errors.As(err, &graphRespErr) {
		if slices.Contains(skipRetryCodes, graphRespErr.GetStatusCode()) {
			return fmt.Errorf("%w (%w)", err, asynq.SkipRetry)
		}
	}

	return err
}

// ExtractResourceNameFromID extracts the resource name from an Azure resource ID.
// Azure resource IDs have the format:
// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{namespace}/{resourceType}/{resourceName}
func ExtractResourceNameFromID(resourceID string) string {
	if resourceID == "" {
		return ""
	}

	parts := strings.Split(strings.TrimSuffix(resourceID, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

// ExtractParentResourceNameFromID extracts the parent resource name from an Azure resource ID.
// This is useful for nested resources like subnets, where you need the VNet name.
// For a subnet ID like: /subscriptions/.../virtualNetworks/{vnetName}/subnets/{subnetName}
// This returns {vnetName}
func ExtractParentResourceNameFromID(resourceID string) string {
	if resourceID == "" {
		return ""
	}

	parts := strings.Split(strings.TrimSuffix(resourceID, "/"), "/")
	if len(parts) >= 3 {
		return parts[len(parts)-3]
	}

	return ""
}
