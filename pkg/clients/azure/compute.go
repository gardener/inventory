// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"

	"github.com/gardener/inventory/pkg/core/registry"
)

// VirtualMachinesClientset provides the registry of Azure Compute API clients
// for interfacing with Virtual Machines.
var VirtualMachinesClientset = registry.New[string, *Client[*armcompute.VirtualMachinesClient]]()
