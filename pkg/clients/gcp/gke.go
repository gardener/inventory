// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	container "cloud.google.com/go/container/apiv1"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ClusterManagerClientset provides the registry of GKE API clients for
// interfacing with the GKE APIs.
var ClusterManagerClientset = registry.New[string, *Client[*container.ClusterManagerClient]]()
