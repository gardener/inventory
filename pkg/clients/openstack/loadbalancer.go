// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gophercloud/gophercloud/v2"

	"github.com/gardener/inventory/pkg/core/registry"
)

// LoadBalancerClientset provides the registry of OpenStack LoadBalancer API clients
// for interfacing with load balancer resources.
var LoadBalancerClientset = registry.New[string, Client[*gophercloud.ServiceClient]]()
