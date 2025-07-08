// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log/slog"

	vaultclients "github.com/gardener/inventory/pkg/clients/vault"
	"github.com/gardener/inventory/pkg/core/config"
	apiclient "github.com/gardener/inventory/pkg/vault/client"
)

// configureVaultClients creates Vault API clients.
func configureVaultClients(ctx context.Context, conf *config.Config) error {
	if !conf.Vault.IsEnabled {
		slog.Warn("Vault is not enabled, will not create API clients")

		return nil
	}

	slog.Info("configuring vault clients")
	for name, serverConfig := range conf.Vault.Servers {
		c, err := apiclient.NewFromConfig(&serverConfig)
		if err != nil {
			return fmt.Errorf("vault: cannot configure client for %s: %s", name, err)
		}

		if err := c.ManageAuthTokenLifetime(ctx); err != nil {
			return fmt.Errorf("vault: cannot start managing auth token lifetime for %s: %s", name, err)
		}

		vaultclients.Clientset.Overwrite(name, c)
		slog.Info(
			"configured vault client",
			"name", name,
			"address", c.Address(),
		)
	}

	return nil
}
