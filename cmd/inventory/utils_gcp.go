// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"slices"

	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/version"
)

// errNoGCPAuthenticationMethod is an error, which is returned when using an
// unknown/unsupported GCP authentication method.
var errNoGCPAuthenticationMethod = errors.New("no GCP authentication method specified")

// errUnknownGCPAuthenticationMethod is an error, which is returned when using
// an unknown/unsupported GCP authentication method/strategy.
var errUnknownGCPAuthenticationMethod = errors.New("unknown GCP authentication method specified")

// validateGCPConfig validates the GCP configuration settings.
func validateGCPConfig(conf *config.Config) error {
	if conf.GCP.UserAgent == "" {
		conf.GCP.UserAgent = fmt.Sprintf("gardener-inventory/%s", version.Version)
	}

	// Make sure that the GCP services have named credentials configured.
	services := map[string][]string{
		"resource_manager": conf.GCP.Services.ResourceManager.UseCredentials,
		"compute":          conf.GCP.Services.Compute.UseCredentials,
	}

	for service, namedCredentials := range services {
		// We expect named credentials to be specified explicitly
		if len(namedCredentials) == 0 {
			return fmt.Errorf("gcp: %w: %s", errNoServiceCredentials, service)
		}

		// Validate that the named credentials are actually defined.
		for _, nc := range namedCredentials {
			if _, ok := conf.GCP.Credentials[nc]; !ok {
				return fmt.Errorf("gcp: %w service %s refers to %s", errUnknownNamedCredentials, service, nc)
			}
		}
	}

	// Validate the named credentials for using valid authentication
	// methods/strategies.
	supportedAuthnMethods := []string{
		config.GCPAuthenticationMethodNone,
		config.GCPAuthenticationMethodKeyFile,
	}

	for name, creds := range conf.GCP.Credentials {
		if creds.Authentication == "" {
			return fmt.Errorf("%w: %s", errNoGCPAuthenticationMethod, name)
		}
		if !slices.Contains(supportedAuthnMethods, creds.Authentication) {
			return fmt.Errorf("%w: %s uses %s", errUnknownGCPAuthenticationMethod, name, creds.Authentication)
		}
	}

	return nil
}
