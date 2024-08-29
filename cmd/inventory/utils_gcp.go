// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	compute "cloud.google.com/go/compute/apiv1"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"google.golang.org/api/option"

	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/version"
)

// errNoGCPAuthenticationMethod is an error, which is returned when using an
// unknown/unsupported GCP authentication method.
var errNoGCPAuthenticationMethod = errors.New("no GCP authentication method specified")

// errUnknownGCPAuthenticationMethod is an error, which is returned when using
// an unknown/unsupported GCP authentication method/strategy.
var errUnknownGCPAuthenticationMethod = errors.New("unknown GCP authentication method specified")

// errNoGCPKeyFile is an error, which is returned when no path to a service
// account JSON Key File was specified for a named credential.
var errNoGCPKeyFile = errors.New("no service account JSON key file specified")

// errNoGCPProjects is an error, which is returned when named credentials are
// configured without specifying associated projects.
var errNoGCPProjects = errors.New("no GCP projects specified")

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
				return fmt.Errorf("gcp: %w: service %s refers to %s", errUnknownNamedCredentials, service, nc)
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
			return fmt.Errorf("gcp: %w: credentials %s", errNoGCPAuthenticationMethod, name)
		}
		if !slices.Contains(supportedAuthnMethods, creds.Authentication) {
			return fmt.Errorf("gcp: %w: %s uses %s", errUnknownGCPAuthenticationMethod, name, creds.Authentication)
		}
		if len(creds.Projects) == 0 {
			return fmt.Errorf("gcp: %w: credentials %s", errNoGCPProjects, name)
		}
	}

	return nil
}

// getGCPClientOptions returns the slice of [option.ClientOption], which are
// derived from the configured named credentials settings.
func getGCPClientOptions(conf *config.Config, namedCredentials string) ([]option.ClientOption, error) {
	creds, ok := conf.GCP.Credentials[namedCredentials]
	if !ok {
		return nil, fmt.Errorf("gcp: %w: %s", errUnknownNamedCredentials, namedCredentials)
	}

	// Default set of options
	opts := []option.ClientOption{
		option.WithUserAgent(conf.GCP.UserAgent),
	}

	switch creds.Authentication {
	case config.GCPAuthenticationMethodNone:
		// Load Application Default Credentials only, nothing to be done
		// from our side.
		break
	case config.GCPAuthenticationMethodKeyFile:
		// JSON Key file authentication
		if creds.KeyFile.Path == "" {
			return nil, fmt.Errorf("gcp: %w: credentials %s", errNoGCPKeyFile, namedCredentials)
		}
		opts = append(opts, option.WithCredentialsFile(creds.KeyFile.Path))
	default:
		return nil, fmt.Errorf("gcp: %w: %s uses %s", errUnknownGCPAuthenticationMethod, namedCredentials, creds.Authentication)
	}

	return opts, nil
}

// configureGCPResourceManagerClientsets configures the GCP Resource Manager API
// clientsets.
func configureGCPResourceManagerClientsets(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.GCP.Services.ResourceManager.UseCredentials {
		opts, err := getGCPClientOptions(conf, namedCreds)
		if err != nil {
			return err
		}

		nc, ok := conf.GCP.Credentials[namedCreds]
		if !ok {
			return fmt.Errorf("gcp: %w: %s", errUnknownNamedCredentials, namedCreds)
		}

		// Register the client for each specified GCP project
		for _, project := range nc.Projects {
			c, err := resourcemanager.NewProjectsRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create client for %s: %w", namedCreds, err)
			}
			client := &gcpclients.Client[*resourcemanager.ProjectsClient]{
				NamedCredentials: namedCreds,
				ProjectID:        project,
				Client:           c,
			}
			gcpclients.ProjectsClientset.Overwrite(project, client)
			slog.Info(
				"configured GCP client",
				"service", "resource_manager",
				"sub_service", "projects",
				"credentials", client.NamedCredentials,
				"project", project,
			)
		}
	}

	return nil
}

// configureGCPComputeClientsets configures the GCP Compute API clientsets.
func configureGCPComputeClientsets(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.GCP.Services.Compute.UseCredentials {
		opts, err := getGCPClientOptions(conf, namedCreds)
		if err != nil {
			return err
		}

		nc, ok := conf.GCP.Credentials[namedCreds]
		if !ok {
			return fmt.Errorf("gcp: %w: %s", errUnknownNamedCredentials, namedCreds)
		}

		// Register the client for each specified GCP project
		for _, project := range nc.Projects {
			// Instances
			instanceClient, err := compute.NewInstancesRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create gcp instance client for %s: %w", namedCreds, err)
			}
			instanceClientWrapper := &gcpclients.Client[*compute.InstancesClient]{
				NamedCredentials: namedCreds,
				ProjectID:        project,
				Client:           instanceClient,
			}
			gcpclients.InstancesClientset.Overwrite(project, instanceClientWrapper)

			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "instances",
				"credentials", namedCreds,
				"project", project,
			)

			// VPCs
			networkClient, err := compute.NewNetworksRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create gcp network client for %s: %w", namedCreds, err)
			}

			networkClientWrapper := &gcpclients.Client[*compute.NetworksClient]{
				NamedCredentials: namedCreds,
				ProjectID:        project,
				Client:           networkClient,
			}
			gcpclients.NetworksClientset.Overwrite(project, networkClientWrapper)

			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "networks",
				"credentials", namedCreds,
				"project", project,
			)

			// Global & Regional Addresses
			addrClient, err := compute.NewAddressesRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create addresses client for %s: %w", namedCreds, err)
			}

			addrClientWrapper := &gcpclients.Client[*compute.AddressesClient]{
				NamedCredentials: namedCreds,
				ProjectID:        project,
				Client:           addrClient,
			}
			gcpclients.AddressesClientset.Overwrite(project, addrClientWrapper)

			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "addresses",
				"credentials", namedCreds,
				"project", project,
			)

			globalAddrClient, err := compute.NewGlobalAddressesRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create global addresses client for %s: %w", namedCreds, err)
			}

			globalAddrClientWrapper := &gcpclients.Client[*compute.GlobalAddressesClient]{
				NamedCredentials: namedCreds,
				ProjectID:        project,
				Client:           globalAddrClient,
			}
			gcpclients.GlobalAddressesClientset.Overwrite(project, globalAddrClientWrapper)

			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "global_addresses",
				"credentials", namedCreds,
				"project", project,
			)


			subnetClient, err := compute.NewSubnetworksRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create gcp subnet client for %s: %w", namedCreds, err)
			}

			subnetClientWrapper := &gcpclients.Client[*compute.SubnetworksClient]{
				NamedCredentials: namedCreds,
				ProjectID:        project,
				Client:           subnetClient,
			}
			gcpclients.SubnetworksClientset.Overwrite(project, subnetClientWrapper)

			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "subnetworks",
				"credentials", namedCreds,
				"project", project,
			)
		}
	}

	return nil
}

// configureGCPClients creates the GCP API clients from the specified
// configuration.
func configureGCPClients(ctx context.Context, conf *config.Config) error {
	if !conf.GCP.IsEnabled {
		slog.Warn("GCP is not enabled, will not create API clients")
		return nil
	}

	slog.Info("configuring GCP clients")
	configFuncs := map[string]func(ctx context.Context, conf *config.Config) error{
		"resource_manager": configureGCPResourceManagerClientsets,
		"compute":          configureGCPComputeClientsets,
	}

	for svc, configFunc := range configFuncs {
		if err := configFunc(ctx, conf); err != nil {
			return fmt.Errorf("unable to configure GCP clients for %s: %w", svc, err)
		}
	}

	return nil
}

// closeGCPClients closes the existing GCP client connections
func closeGCPClients() {
	_ = gcpclients.ProjectsClientset.Range(func(_ string, client *gcpclients.Client[*resourcemanager.ProjectsClient]) error {
		client.Client.Close()
		return nil
	})

	_ = gcpclients.InstancesClientset.Range(func(_ string, client *gcpclients.Client[*compute.InstancesClient]) error {
		client.Client.Close()
		return nil
	})

	_ = gcpclients.NetworksClientset.Range(func(_ string, client *gcpclients.Client[*compute.NetworksClient]) error {
		client.Client.Close()
		return nil
	})

	_ = gcpclients.AddressesClientset.Range(func(_ string, client *gcpclients.Client[*compute.AddressesClient]) error {
		client.Client.Close()
		return nil
	})

	_ = gcpclients.GlobalAddressesClientset.Range(func(_ string, client *gcpclients.Client[*compute.GlobalAddressesClient]) error {
		client.Client.Close()
		return nil
	})

	_ = gcpclients.SubnetworksClientset.Range(func(_ string, client *gcpclients.Client[*compute.SubnetworksClient]) error {
		client.Client.Close()
		return nil
	})
}
