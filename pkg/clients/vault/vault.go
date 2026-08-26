// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package vault

import (
	"github.com/gardener/inventory/pkg/core/registry"
	apiclient "github.com/gardener/inventory/pkg/vault/client"
)

// Clientset provides the registry of Vault API clients, which are used by
// workers during runtime.
var Clientset = registry.New[string, *apiclient.Client]()
