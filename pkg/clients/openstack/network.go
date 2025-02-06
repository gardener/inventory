// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gophercloud/gophercloud/v2"
)

// NetworkClientset provides the registry of OpenStack Network API clients
// for interfacing with network resoures.
var NetworkClientset = registry.New[string, Client[*gophercloud.ServiceClient]]()
