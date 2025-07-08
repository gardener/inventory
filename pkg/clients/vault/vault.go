// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
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
