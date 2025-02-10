// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gophercloud/gophercloud/v2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// BlockStorageClientset provides the registry of OpenStack Block Storage API clients
// for interfacing with block storage resources.
var BlockStorageClientset = registry.New[string, Client[*gophercloud.ServiceClient]]()
