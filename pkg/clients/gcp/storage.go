// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"cloud.google.com/go/storage"

	"github.com/gardener/inventory/pkg/core/registry"
)

// StorageClientset provides the registry of GCP API clients for interfacing
// with the storage API service.
var StorageClientset = registry.New[string, *Client[*storage.Client]]()
