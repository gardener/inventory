// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	gophercloudconfig "github.com/gophercloud/gophercloud/v2/openstack/config"

	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/core/config"
)

var errNoUsername = errors.New("no username specified")
var errNoPasswordFile = errors.New("no password file specified")
var errNoAppCredentialsID = errors.New("no app credentials id specified")
var errNoAppCredentialsSecretFile = errors.New("no app credentials secret file specified")
var errNoAuthEndpoint = errors.New("no authentication endpoint specified")
var errNoDomain = errors.New("no domain specified")
var errNoRegion = errors.New("no region specified")
var errNoProject = errors.New("no project specified")
var errNoProjectID = errors.New("no project id specified")

// validateOpenStackConfig validates the OpenStack configuration settings.
func validateOpenStackConfig(conf *config.Config) error {
	// Make sure that the services have named credentials configured.
	services := map[string][]config.OpenStackServiceConfig{
		"compute":       conf.OpenStack.Services.Compute,
		"network":       conf.OpenStack.Services.Network,
		"block_storage": conf.OpenStack.Services.BlockStorage,
	}

	for service, serviceConfigs := range services {
		if len(serviceConfigs) == 0 {
			continue
		}

		// Validate that the named credentials are actually defined.
		for _, config := range serviceConfigs {
			namedCreds := config.UseCredentials
			if namedCreds == "" {
				return fmt.Errorf("OpenStack: %w: %s", errNoServiceCredentials, service)
			}

			if _, ok := conf.OpenStack.Credentials[namedCreds]; !ok {
				return fmt.Errorf("OpenStack: %w: service %s refers to %s", errUnknownNamedCredentials, service, namedCreds)
			}

			if config.AuthEndpoint == "" {
				return fmt.Errorf("OpenStack: %w: %s", errNoAuthEndpoint, service)
			}

			if config.Domain == "" {
				return fmt.Errorf("OpenStack: %w: %s", errNoDomain, service)
			}

			if config.Region == "" {
				return fmt.Errorf("OpenStack: %w: %s", errNoRegion, service)
			}

			if config.Project == "" {
				return fmt.Errorf("OpenStack: %w: %s", errNoProject, service)
			}

			if config.ProjectID == "" {
				return fmt.Errorf("OpenStack: %w: %s", errNoProjectID, service)
			}
		}

		for name, creds := range conf.OpenStack.Credentials {
			if creds.Authentication == "" {
				return fmt.Errorf("OpenStack: %w: credentials %s", errNoAuthenticationMethod, name)
			}

			switch creds.Authentication {
			case config.OpenStackAuthenticationMethodPassword:
				if creds.Password.Username == "" {
					return fmt.Errorf("OpenStack: %w: %s", errNoUsername, name)
				}
				if creds.Password.PasswordFile == "" {
					return fmt.Errorf("OpenStack: %w: %s", errNoPasswordFile, name)
				}
			case config.OpenStackAuthenticationMethodAppCredentials:
				if creds.AppCredentials.AppCredentialsID == "" {
					return fmt.Errorf("OpenStack: %w: %s", errNoAppCredentialsID, name)
				}
				if creds.AppCredentials.AppCredentialsSecretFile == "" {
					return fmt.Errorf("OpenStack: %w: %s", errNoAppCredentialsSecretFile, name)
				}
			default:
				return fmt.Errorf("OpenStack: %w: %s uses %s", errUnknownAuthenticationMethod, name, creds.Authentication)
			}
		}
	}

	return nil
}

// configureOpenStackClients creates the OpenStack API clients from the specified
// configuration.
func configureOpenStackClients(ctx context.Context, conf *config.Config) error {
	if !conf.OpenStack.IsEnabled {
		slog.Warn("OpenStack is not enabled, will not create API clients")
		return nil
	}

	slog.Info("configuring OpenStack clients")
	if err := validateOpenStackConfig(conf); err != nil {
		return fmt.Errorf("invalid OpenStack configuration: %w", err)
	}

	configFuncs := map[string]func(ctx context.Context, conf *config.Config) error{
		"compute":       configureOpenStackComputeClientsets,
		"network":       configureOpenStackNetworkClientsets,
		"block_storage": configureOpenStackBlockStorageClientsets,
	}

	if conf.Debug {
		if err := os.Setenv("OS_DEBUG", "all"); err != nil {
			return err
		}
	}

	for svc, configFunc := range configFuncs {
		if err := configFunc(ctx, conf); err != nil {
			return fmt.Errorf("unable to configure OpenStack clients for %s: %w", svc, err)
		}
	}

	return nil
}

func newOpenStackProviderClient(
	ctx context.Context,
	clientConfig *config.OpenStackServiceConfig,
	creds config.OpenStackCredentialsConfig,
) (*gophercloud.ProviderClient, error) {
	var authOpts gophercloud.AuthOptions

	switch creds.Authentication {
	case config.OpenStackAuthenticationMethodPassword:
		username := strings.TrimSpace(creds.Password.Username)
		if username == "" {
			return nil, fmt.Errorf("no username specified for project %s", clientConfig.Project)
		}

		rawPassword, err := os.ReadFile(creds.Password.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read password file: %w", err)
		}
		password := strings.TrimSpace(string(rawPassword))
		if password == "" {
			return nil, fmt.Errorf("no password specified for project %s", clientConfig.Project)
		}

		authOpts = gophercloud.AuthOptions{
			IdentityEndpoint: clientConfig.AuthEndpoint,
			DomainName:       clientConfig.Domain,
			TenantName:       clientConfig.Project,
			Username:         username,
			Password:         password,
		}
	case config.OpenStackAuthenticationMethodAppCredentials:
		appID := strings.TrimSpace(creds.AppCredentials.AppCredentialsID)
		if appID == "" {
			return nil, fmt.Errorf("no app credentials id specified for project %s", clientConfig.Project)
		}

		rawAppSecret, err := os.ReadFile(creds.AppCredentials.AppCredentialsSecretFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read app credentials secret file: %w", err)
		}
		appSecret := strings.TrimSpace(string(rawAppSecret))

		if appSecret == "" {
			return nil, fmt.Errorf("no app credentials secret specified for project %s", clientConfig.Project)
		}

		authOpts = gophercloud.AuthOptions{
			IdentityEndpoint:            clientConfig.AuthEndpoint,
			ApplicationCredentialID:     appID,
			ApplicationCredentialSecret: appSecret,
		}
	default:
		return nil, fmt.Errorf("unknown authentication method: %s", creds.Authentication)
	}

	return gophercloudconfig.NewProviderClient(ctx, authOpts)
}

// configureOpenStackComputeClientsets configures the OpenStack Compute API clientsets.
func configureOpenStackComputeClientsets(ctx context.Context, conf *config.Config) error {
	for _, clientConfig := range conf.OpenStack.Services.Compute {
		creds := clientConfig.UseCredentials
		namedCreds := conf.OpenStack.Credentials[creds]

		providerClient, err := newOpenStackProviderClient(ctx, &clientConfig, namedCreds)

		if err != nil {
			return fmt.Errorf("unable to create client for service with credentials %s: %w", creds, err)
		}

		computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
			Region: clientConfig.Region,
		})

		if err != nil {
			return fmt.Errorf("unable to create client for compute service with credentials %s: %w", creds, err)
		}

		client := openstackclients.Client[*gophercloud.ServiceClient]{
			NamedCredentials: creds,
			ProjectID:        clientConfig.ProjectID,
			Region:           clientConfig.Region,
			Domain:           clientConfig.Domain,
			Client:           computeClient,
		}
		openstackclients.ComputeClientset.Overwrite(
			clientConfig.ProjectID,
			client,
		)

		slog.Info(
			"configured OpenStack client",
			"service", "compute",
			"credentials", creds,
			"region", clientConfig.Region,
			"domain", clientConfig.Domain,
			"project", clientConfig.Project,
			"auth_endpoint", clientConfig.AuthEndpoint,
			"auth_method", namedCreds.Authentication,
		)
	}
	return nil
}

// configureOpenStackNetworkClientsets configures the OpenStack Network API clientsets.
func configureOpenStackNetworkClientsets(ctx context.Context, conf *config.Config) error {
	for _, clientConfig := range conf.OpenStack.Services.Network {
		creds := clientConfig.UseCredentials
		namedCreds := conf.OpenStack.Credentials[creds]

		providerClient, err := newOpenStackProviderClient(ctx, &clientConfig, namedCreds)

		if err != nil {
			return fmt.Errorf("unable to create client for service with credentials %s: %w", creds, err)
		}

		networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{
			Region: clientConfig.Region,
		})

		if err != nil {
			return fmt.Errorf("unable to create client for network service with credentials %s: %w", creds, err)
		}

		client := openstackclients.Client[*gophercloud.ServiceClient]{
			NamedCredentials: creds,
			ProjectID:        clientConfig.ProjectID,
			Region:           clientConfig.Region,
			Domain:           clientConfig.Domain,
			Client:           networkClient,
		}
		openstackclients.NetworkClientset.Overwrite(
			clientConfig.ProjectID,
			client,
		)

		slog.Info(
			"configured OpenStack client",
			"service", "network",
			"credentials", creds,
			"region", clientConfig.Region,
			"domain", clientConfig.Domain,
			"project", clientConfig.Project,
			"auth_endpoint", clientConfig.AuthEndpoint,
			"auth_method", namedCreds.Authentication,
		)
	}
	return nil
}

// configureOpenStackBlockStorageClientsets configures the OpenStack Block Storage API clientsets.
func configureOpenStackBlockStorageClientsets(ctx context.Context, conf *config.Config) error {
	for _, clientConfig := range conf.OpenStack.Services.BlockStorage {
		creds := clientConfig.UseCredentials
		namedCreds := conf.OpenStack.Credentials[creds]

		providerClient, err := newOpenStackProviderClient(ctx, &clientConfig, namedCreds)

		if err != nil {
			return fmt.Errorf("unable to create client for service with credentials %s: %w", creds, err)
		}

		blockStorageClient, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{
			Region: clientConfig.Region,
		})

		if err != nil {
			return fmt.Errorf("unable to create client for block storage service with credentials %s: %w", creds, err)
		}

		client := openstackclients.Client[*gophercloud.ServiceClient]{
			NamedCredentials: creds,
			ProjectID:        clientConfig.ProjectID,
			Region:           clientConfig.Region,
			Domain:           clientConfig.Domain,
			Client:           blockStorageClient,
		}
		openstackclients.BlockStorageClientset.Overwrite(
			clientConfig.ProjectID,
			client,
		)

		slog.Info(
			"configured OpenStack client",
			"service", "block_storage",
			"credentials", creds,
			"region", clientConfig.Region,
			"domain", clientConfig.Domain,
			"project", clientConfig.Project,
			"auth_endpoint", clientConfig.AuthEndpoint,
			"auth_method", namedCreds.Authentication,
		)
	}
	return nil
}
