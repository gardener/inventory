// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	"github.com/gardener/inventory/pkg/core/registry"
)

// PublicIPAddressesClientset provides the registry of Azure API clients
// for interfacing with Public IP Addresses.
var PublicIPAddressesClientset = registry.New[string, *Client[*armnetwork.PublicIPAddressesClient]]()
