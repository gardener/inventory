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
