// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gophercloud/gophercloud/v2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ComputeClientset provides the registry of OpenStack Compute API clients
// for interfacing with compute resources (servers, etc).
var ComputeClientset = registry.New[string, Client[*gophercloud.ServiceClient]]()
