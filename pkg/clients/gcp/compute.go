// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	compute "cloud.google.com/go/compute/apiv1"

	"github.com/gardener/inventory/pkg/core/registry"
)

// InstancesClientset provides the registry of GCP API clients for interfacing
// with the Compute Instances API service.
var InstancesClientset = registry.New[string, *Client[*compute.InstancesClient]]()

// NetworksClientset provides the registry of GCP API clients for interfacing
// with the networks API service.
var NetworksClientset = registry.New[string, *Client[*compute.NetworksClient]]()

// AddressesClientset provides the registry of GCP API clients for interfacing
// with the Compute Addresses API service.
var AddressesClientset = registry.New[string, *Client[*compute.AddressesClient]]()

// GlobalAddressesClientset provides the registry of GCP API clients for
// interfacing with the Compute Global Addresses API service.
var GlobalAddressesClientset = registry.New[string, *Client[*compute.GlobalAddressesClient]]()

// SubnetworksClientset provides the registry of GCP API clients for interfacing
// with the subnet API service.
var SubnetworksClientset = registry.New[string, *Client[*compute.SubnetworksClient]]()

// DisksClientset provides the registry of GCP API clients for interfacing
// with the disk API service.
var DisksClientset = registry.New[string, *Client[*compute.DisksClient]]()
