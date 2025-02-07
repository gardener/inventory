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

	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	"github.com/gardener/inventory/pkg/core/config"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	gophercloudconfig "github.com/gophercloud/gophercloud/v2/openstack/config"
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

// configureOpenStackComputeClientsets configures the OpenStack Compute API clientsets.
func configureOpenStackComputeClientsets(ctx context.Context, conf *config.Config) error {
	for _, clientConfig := range conf.OpenStack.Services.Compute {
		domain := clientConfig.Domain
		region := clientConfig.Region
		project := clientConfig.Project
		projectID := clientConfig.ProjectID
		authEndpoint := clientConfig.AuthEndpoint

		cred := clientConfig.UseCredentials
		namedCreds := conf.OpenStack.Credentials[cred]

		var authOpts gophercloud.AuthOptions
		switch namedCreds.Authentication {
		case config.OpenStackAuthenticationMethodPassword:
			username := strings.TrimSpace(namedCreds.Password.Username)

			rawPassword, err := os.ReadFile(namedCreds.Password.PasswordFile)
			password := strings.TrimSpace(string(rawPassword))

			if err != nil {
				return fmt.Errorf("unable to read password file for service %s: %w", cred, err)
			}

			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint: authEndpoint,
				DomainName:       domain,
				TenantName:       project,
				Username:         username,
				Password:         password,
			}
		case config.OpenStackAuthenticationMethodAppCredentials:
			appID := strings.TrimSpace(namedCreds.AppCredentials.AppCredentialsID)

			rawAppSecret, err := os.ReadFile(namedCreds.AppCredentials.AppCredentialsSecretFile)
			if err != nil {
				return fmt.Errorf("unable to read app credentials secret file: %w", err)
			}
			appSecret := strings.TrimSpace(string(rawAppSecret))

			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint:            authEndpoint,
				ApplicationCredentialID:     appID,
				ApplicationCredentialSecret: appSecret,
			}
		default:
			return fmt.Errorf("unknown authentication method: %s", namedCreds.Authentication)
		}

		providerClient, err := gophercloudconfig.NewProviderClient(ctx, authOpts)
		if err != nil {
			return fmt.Errorf("unable to create client for service with credentials %s: %w", cred, err)
		}

		computeClient, err := openstack.NewComputeV2(providerClient, gophercloud.EndpointOpts{
			Region: region,
		})

		if err != nil {
			return fmt.Errorf("unable to create client for compute service with credentials %s: %w", cred, err)
		}

		client := openstackclients.Client[*gophercloud.ServiceClient]{
			NamedCredentials: cred,
			ProjectID:        projectID,
			Region:           region,
			Domain:           domain,
			Client:           computeClient,
		}
		openstackclients.ComputeClientset.Overwrite(
			projectID,
			client,
		)

		slog.Info(
			"configured OpenStack client",
			"service", "compute",
			"credentials", cred,
			"region", region,
			"domain", domain,
			"project", project,
			"auth_endpoint", authEndpoint,
			"auth_method", namedCreds.Authentication,
		)
	}
	return nil
}

// configureOpenStackNetworkClientsets configures the OpenStack Network API clientsets.
func configureOpenStackNetworkClientsets(ctx context.Context, conf *config.Config) error {
	for _, clientConfig := range conf.OpenStack.Services.Network {
		domain := clientConfig.Domain
		region := clientConfig.Region
		project := clientConfig.Project
		projectID := clientConfig.ProjectID
		authEndpoint := clientConfig.AuthEndpoint

		cred := clientConfig.UseCredentials
		namedCreds := conf.OpenStack.Credentials[cred]

		var authOpts gophercloud.AuthOptions
		switch namedCreds.Authentication {
		case config.OpenStackAuthenticationMethodPassword:
			username := strings.TrimSpace(namedCreds.Password.Username)

			rawPassword, err := os.ReadFile(namedCreds.Password.PasswordFile)
			password := strings.TrimSpace(string(rawPassword))

			if err != nil {
				return fmt.Errorf("unable to read password file for service %s: %w", cred, err)
			}

			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint: authEndpoint,
				DomainName:       domain,
				TenantName:       project,
				Username:         username,
				Password:         password,
			}
		case config.OpenStackAuthenticationMethodAppCredentials:
			appID := strings.TrimSpace(namedCreds.AppCredentials.AppCredentialsID)

			rawAppSecret, err := os.ReadFile(namedCreds.AppCredentials.AppCredentialsSecretFile)
			if err != nil {
				return fmt.Errorf("unable to read app credentials secret file: %w", err)
			}
			appSecret := string(rawAppSecret)

			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint:            authEndpoint,
				ApplicationCredentialID:     appID,
				ApplicationCredentialSecret: appSecret,
			}
		default:
			return fmt.Errorf("unknown authentication method: %s", namedCreds.Authentication)
		}

		providerClient, err := gophercloudconfig.NewProviderClient(ctx, authOpts)
		if err != nil {
			return fmt.Errorf("unable to create client for service with credentials %s: %w", cred, err)
		}

		networkClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})

		if err != nil {
			return fmt.Errorf("unable to create client for network service with credentials %s: %w", cred, err)
		}

		client := openstackclients.Client[*gophercloud.ServiceClient]{
			NamedCredentials: cred,
			ProjectID:        projectID,
			Region:           region,
			Domain:           domain,
			Client:           networkClient,
		}
		openstackclients.NetworkClientset.Overwrite(
			projectID,
			client,
		)

		slog.Info(
			"configured OpenStack client",
			"service", "network",
			"credentials", cred,
			"region", region,
			"domain", domain,
			"project", project,
			"auth_endpoint", authEndpoint,
			"auth_method", namedCreds.Authentication,
		)
	}
	return nil
}

// configureOpenStackBlockStorageClientsets configures the OpenStack Block Storage API clientsets.
func configureOpenStackBlockStorageClientsets(ctx context.Context, conf *config.Config) error {
	for _, clientConfig := range conf.OpenStack.Services.BlockStorage {
		domain := clientConfig.Domain
		region := clientConfig.Region
		project := clientConfig.Project
		projectID := clientConfig.ProjectID
		authEndpoint := clientConfig.AuthEndpoint

		cred := clientConfig.UseCredentials
		namedCreds := conf.OpenStack.Credentials[cred]

		var authOpts gophercloud.AuthOptions
		switch namedCreds.Authentication {
		case config.OpenStackAuthenticationMethodPassword:
			username := strings.TrimSpace(namedCreds.Password.Username)

			rawPassword, err := os.ReadFile(namedCreds.Password.PasswordFile)
			password := strings.TrimSpace(string(rawPassword))

			if err != nil {
				return fmt.Errorf("unable to read password file for service %s: %w", cred, err)
			}

			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint: authEndpoint,
				DomainName:       domain,
				TenantName:       project,
				Username:         username,
				Password:         password,
			}
		case config.OpenStackAuthenticationMethodAppCredentials:
			appID := strings.TrimSpace(namedCreds.AppCredentials.AppCredentialsID)

			rawAppSecret, err := os.ReadFile(namedCreds.AppCredentials.AppCredentialsSecretFile)
			if err != nil {
				return fmt.Errorf("unable to read app credentials secret file: %w", err)
			}
			appSecret := string(rawAppSecret)

			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint:            authEndpoint,
				ApplicationCredentialID:     appID,
				ApplicationCredentialSecret: appSecret,
			}
		default:
			return fmt.Errorf("unknown authentication method: %s", namedCreds.Authentication)
		}

		providerClient, err := gophercloudconfig.NewProviderClient(ctx, authOpts)
		if err != nil {
			return fmt.Errorf("unable to create client for service with credentials %s: %w", cred, err)
		}

		blockStorageClient, err := openstack.NewBlockStorageV3(providerClient, gophercloud.EndpointOpts{
			Region: region,
		})

		if err != nil {
			return fmt.Errorf("unable to create client for block storage service with credentials %s: %w", cred, err)
		}

		client := openstackclients.Client[*gophercloud.ServiceClient]{
			NamedCredentials: cred,
			ProjectID:        projectID,
			Region:           region,
			Domain:           domain,
			Client:           blockStorageClient,
		}
		openstackclients.BlockStorageClientset.Overwrite(
			projectID,
			client,
		)

		slog.Info(
			"configured OpenStack client",
			"service", "block_storage",
			"credentials", cred,
			"region", region,
			"domain", domain,
			"project", project,
			"auth_endpoint", authEndpoint,
			"auth_method", namedCreds.Authentication,
		)
	}
	return nil
}
