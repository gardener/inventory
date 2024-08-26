// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ProjectsClientset provides the registry of GCP API clients for interfacing
// with the Cloud Resource Manager Projects API service.
var ProjectsClientset = registry.New[string, *Client[*resourcemanager.ProjectsClient]]()
