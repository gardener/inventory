// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"

	"github.com/gardener/inventory/pkg/core/registry"
)

// GraphClientset provides the registry of Graph API clients.
var GraphClientset = registry.New[string, *Client[*msgraphsdk.GraphServiceClient]]()
