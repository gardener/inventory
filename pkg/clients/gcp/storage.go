// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
