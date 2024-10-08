// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"

	"github.com/gardener/inventory/pkg/core/registry"
)

// SubscriptionsClientset provides the registry of Azure API clients for
// interfacing with Subscriptions.
var SubscriptionsClientset = registry.New[string, *Client[*armsubscription.SubscriptionsClient]]()

// ResourceGroupsClientset provides the registry of Azure API clients for
// interfacing with Resource Groups.
var ResourceGroupsClientset = registry.New[string, *Client[*armresources.ResourceGroupsClient]]()
