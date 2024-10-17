// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	armstorage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/gardener/inventory/pkg/core/registry"
)

// BlobContainersClientset provides the registry of Azure API clients
// for interfacing with Blog containers.
var BlobContainersClientset = registry.New[string, *Client[*armstorage.BlobContainersClient]]()

// StorageAccountsClientset provides the registry of Azure API clients
// for interfacing with Storage Accounts.
var StorageAccountsClientset = registry.New[string, *Client[*armstorage.AccountsClient]]()
