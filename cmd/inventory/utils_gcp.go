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
	container "cloud.google.com/go/container/apiv1"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/version"
)

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
		"resource_manager":  conf.GCP.Services.ResourceManager.UseCredentials,
		"compute":           conf.GCP.Services.Compute.UseCredentials,
		"storage":           conf.GCP.Services.Storage.UseCredentials,
		"gke":               conf.GCP.Services.GKE.UseCredentials,
		"soil-gcp-regional": {conf.GCP.SoilCluster.UseCredentials},
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
			return fmt.Errorf("gcp: %w: credentials %s", errNoAuthenticationMethod, name)
		}
		if !slices.Contains(supportedAuthnMethods, creds.Authentication) {
			return fmt.Errorf("gcp: %w: %s uses %s", errUnknownAuthenticationMethod, name, creds.Authentication)
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
		break // nolint: revive
	case config.GCPAuthenticationMethodKeyFile:
		// JSON Key file authentication
		if creds.KeyFile.Path == "" {
			return nil, fmt.Errorf("gcp: %w: credentials %s", errNoGCPKeyFile, namedCredentials)
		}
		opts = append(opts, option.WithAuthCredentialsFile(option.ExternalAccount, creds.KeyFile.Path))
	default:
		return nil, fmt.Errorf("gcp: %w: %s uses %s", errUnknownAuthenticationMethod, namedCredentials, creds.Authentication)
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
			gcpclients.ProjectsClientset.Overwrite(
				project,
				&gcpclients.Client[*resourcemanager.ProjectsClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           c,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "resource_manager",
				"sub_service", "projects",
				"credentials", namedCreds,
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
				return fmt.Errorf("gcp: cannot create instance client for %s: %w", namedCreds, err)
			}
			gcpclients.InstancesClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.InstancesClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           instanceClient,
				},
			)
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
				return fmt.Errorf("gcp: cannot create network client for %s: %w", namedCreds, err)
			}
			gcpclients.NetworksClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.NetworksClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           networkClient,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "networks",
				"credentials", namedCreds,
				"project", project,
			)

			// Regional Addresses client
			addrClient, err := compute.NewAddressesRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create addresses client for %s: %w", namedCreds, err)
			}
			gcpclients.AddressesClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.AddressesClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           addrClient,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "addresses",
				"credentials", namedCreds,
				"project", project,
			)

			// Global Addresses client
			globalAddrClient, err := compute.NewGlobalAddressesRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create global addresses client for %s: %w", namedCreds, err)
			}
			gcpclients.GlobalAddressesClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.GlobalAddressesClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           globalAddrClient,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "global-addresses",
				"credentials", namedCreds,
				"project", project,
			)

			// Subnet clients
			subnetClient, err := compute.NewSubnetworksRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create subnet client for %s: %w", namedCreds, err)
			}
			gcpclients.SubnetworksClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.SubnetworksClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           subnetClient,
				},
			)

			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "subnetworks",
				"credentials", namedCreds,
				"project", project,
			)

			// Disk clients
			diskClient, err := compute.NewDisksRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create disk client for %s: %w", namedCreds, err)
			}
			gcpclients.DisksClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.DisksClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           diskClient,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "disks",
				"credentials", namedCreds,
				"project", project,
			)

			// Forwarding Rules clients
			frClient, err := compute.NewForwardingRulesRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create forwarding rules client for %s: %w", namedCreds, err)
			}
			gcpclients.ForwardingRulesClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.ForwardingRulesClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           frClient,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "forwarding-rules",
				"credentials", namedCreds,
				"project", project,
			)

			// Target Pools clients
			tpClient, err := compute.NewTargetPoolsRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create target pools client for %s: %w", namedCreds, err)
			}
			gcpclients.TargetPoolsClientset.Overwrite(
				project,
				&gcpclients.Client[*compute.TargetPoolsClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           tpClient,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "compute",
				"sub_service", "target-pools",
				"credentials", namedCreds,
				"project", project,
			)
		}
	}

	return nil
}

// configureGCPStorageClientsets configures the GCP storage API clientsets.
func configureGCPStorageClientsets(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.GCP.Services.Storage.UseCredentials {
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
			// Buckets
			storageClient, err := storage.NewClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create gcp storage client for %s: %w", namedCreds, err)
			}
			gcpclients.StorageClientset.Overwrite(
				project,
				&gcpclients.Client[*storage.Client]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           storageClient,
				},
			)
			slog.Info(
				"configured GCP client",
				"service", "storage",
				"credentials", namedCreds,
				"project", project,
			)
		}
	}

	return nil
}

// configureGKEClientsets configures the GKE related API clients.
func configureGKEClientsets(ctx context.Context, conf *config.Config) error {
	for _, namedCreds := range conf.GCP.Services.GKE.UseCredentials {
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
			client, err := container.NewClusterManagerRESTClient(ctx, opts...)
			if err != nil {
				return fmt.Errorf("gcp: cannot create gcp cluster manager client for %s: %w", namedCreds, err)
			}
			gcpclients.ClusterManagerClientset.Overwrite(
				project,
				&gcpclients.Client[*container.ClusterManagerClient]{
					NamedCredentials: namedCreds,
					ProjectID:        project,
					Client:           client,
				},
			)

			slog.Info(
				"configured GCP client",
				"service", "cluster_manager",
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
	if err := validateGCPConfig(conf); err != nil {
		return err
	}

	configFuncs := map[string]func(ctx context.Context, conf *config.Config) error{
		"resource_manager": configureGCPResourceManagerClientsets,
		"compute":          configureGCPComputeClientsets,
		"storage":          configureGCPStorageClientsets,
		"gke":              configureGKEClientsets,
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
		return client.Client.Close()
	})

	_ = gcpclients.InstancesClientset.Range(func(_ string, client *gcpclients.Client[*compute.InstancesClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.NetworksClientset.Range(func(_ string, client *gcpclients.Client[*compute.NetworksClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.AddressesClientset.Range(func(_ string, client *gcpclients.Client[*compute.AddressesClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.GlobalAddressesClientset.Range(func(_ string, client *gcpclients.Client[*compute.GlobalAddressesClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.SubnetworksClientset.Range(func(_ string, client *gcpclients.Client[*compute.SubnetworksClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.DisksClientset.Range(func(_ string, client *gcpclients.Client[*compute.DisksClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.StorageClientset.Range(func(_ string, client *gcpclients.Client[*storage.Client]) error {
		return client.Client.Close()
	})

	_ = gcpclients.ForwardingRulesClientset.Range(func(_ string, client *gcpclients.Client[*compute.ForwardingRulesClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.ClusterManagerClientset.Range(func(_ string, client *gcpclients.Client[*container.ClusterManagerClient]) error {
		return client.Client.Close()
	})

	_ = gcpclients.TargetPoolsClientset.Range(func(_ string, client *gcpclients.Client[*compute.TargetPoolsClient]) error {
		return client.Client.Close()
	})
}
