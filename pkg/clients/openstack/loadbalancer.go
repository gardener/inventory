// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gophercloud/gophercloud/v2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// LoadBalancerClientset provides the registry of OpenStack LoadBalancer API clients
// for interfacing with load balancer resources.
var LoadBalancerClientset = registry.New[ClientScope, Client[*gophercloud.ServiceClient]]()
