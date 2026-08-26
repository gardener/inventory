// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gophercloud/gophercloud/v2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// IdentityClientset provides the registry of OpenStack Identity API clients
var IdentityClientset = registry.New[ClientScope, Client[*gophercloud.ServiceClient]]()
