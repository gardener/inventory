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
	"github.com/gardener/inventory/pkg/core/registry"
)

var errNoUsername = errors.New("no username specified")
var errNoPasswordFile = errors.New("no password file specified")
var errNoAppCredentialsID = errors.New("no app credentials id specified")
var errNoAppCredentialsSecretFile = errors.New("no app credentials secret file specified")
var errNoAuthEndpoint = errors.New("no authentication endpoint specified")
var errNoDomain = errors.New("no domain specified")
var errNoRegion = errors.New("no region specified")
var errNoProject = errors.New("no project specified")

// validateOpenStackConfig validates the OpenStack configuration settings.
func validateOpenStackConfig(conf *config.Config) error {
	// Make sure that the services have named credentials configured.
	services := map[string]config.OpenStackServiceCredentials{
		"compute":        conf.OpenStack.Services.Compute,
		"network":        conf.OpenStack.Services.Network,
		"object_storage": conf.OpenStack.Services.ObjectStorage,
		"load_balancer":  conf.OpenStack.Services.LoadBalancer,
		"identity":       conf.OpenStack.Services.Identity,
	}

	for name, creds := range conf.OpenStack.Credentials {
		if creds.AuthEndpoint == "" {
			return fmt.Errorf("OpenStack: %w: credentials %s", errNoAuthEndpoint, name)
		}

		if creds.Domain == "" {
			return fmt.Errorf("OpenStack: %w: credentials %s", errNoDomain, name)
		}

		if creds.Region == "" {
			return fmt.Errorf("OpenStack: %w: credentials %s", errNoRegion, name)
		}

		if creds.Project == "" {
			return fmt.Errorf("OpenStack: %w: credentials %s", errNoProject, name)
		}

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

	for service, serviceCredentials := range services {
		credentials := serviceCredentials.UseCredentials

		if len(credentials) == 0 {
			continue
		}

		// Validate that the named credentials are actually defined.
		for _, cred := range credentials {
			if cred == "" {
				return fmt.Errorf("OpenStack: %w: %s", errNoServiceCredentials, service)
			}

			if _, ok := conf.OpenStack.Credentials[cred]; !ok {
				return fmt.Errorf("OpenStack: %w: service %s refers to %s", errUnknownNamedCredentials, service, cred)
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

	if conf.Debug {
		if err := os.Setenv("OS_DEBUG", "all"); err != nil {
			return err
		}
	}

	slog.Info("configuring OpenStack clients")
	if err := validateOpenStackConfig(conf); err != nil {
		return fmt.Errorf("invalid OpenStack configuration: %w", err)
	}

	configFuncs := map[string]func(ctx context.Context, conf *config.Config) error{
		"compute":        configureOpenStackComputeClientsets,
		"network":        configureOpenStackNetworkClientsets,
		"object_storage": configureOpenStackObjectStorageClientsets,
		"load_balancer":  configureOpenStackLoadBalancerClientsets,
		"identity":       configureOpenStackIdentityClientsets,
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
	creds *config.OpenStackCredentialsConfig,
) (*gophercloud.ProviderClient, error) {
	var authOpts gophercloud.AuthOptions

	switch creds.Authentication {
	case config.OpenStackAuthenticationMethodPassword:
		username := strings.TrimSpace(creds.Password.Username)
		if username == "" {
			return nil, fmt.Errorf("no username specified for project %s", creds.Project)
		}

		rawPassword, err := os.ReadFile(creds.Password.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read password file: %w", err)
		}
		password := strings.TrimSpace(string(rawPassword))
		if password == "" {
			return nil, fmt.Errorf("no password specified for project %s", creds.Project)
		}

		authOpts = gophercloud.AuthOptions{
			IdentityEndpoint: creds.AuthEndpoint,
			DomainName:       creds.Domain,
			TenantName:       creds.Project,
			Username:         username,
			Password:         password,
			AllowReauth:      true,
		}
	case config.OpenStackAuthenticationMethodAppCredentials:
		appID := strings.TrimSpace(creds.AppCredentials.AppCredentialsID)
		if appID == "" {
			return nil, fmt.Errorf("no app credentials id specified for project %s", creds.Project)
		}

		rawAppSecret, err := os.ReadFile(creds.AppCredentials.AppCredentialsSecretFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read app credentials secret file: %w", err)
		}
		appSecret := strings.TrimSpace(string(rawAppSecret))

		if appSecret == "" {
			return nil, fmt.Errorf("no app credentials secret specified for project %s", creds.Project)
		}

		authOpts = gophercloud.AuthOptions{
			IdentityEndpoint:            creds.AuthEndpoint,
			ApplicationCredentialID:     appID,
			ApplicationCredentialSecret: appSecret,
			AllowReauth:                 true,
		}
	default:
		return nil, fmt.Errorf("unknown authentication method: %s", creds.Authentication)
	}

	return gophercloudconfig.NewProviderClient(ctx, authOpts)
}

func configureOpenStackServiceClientset(
	ctx context.Context,
	serviceName string,
	clientset *registry.Registry[openstackclients.ClientScope, openstackclients.Client[*gophercloud.ServiceClient]],
	serviceConfig config.OpenStackServiceCredentials,
	conf *config.Config,
	serviceFunc func(providerClient *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error)) error {
	for _, credentials := range serviceConfig.UseCredentials {
		namedCreds := conf.OpenStack.Credentials[credentials]
		providerClient, err := newOpenStackProviderClient(ctx, &namedCreds)

		if err != nil {
			return fmt.Errorf("unable to create client for service with credentials %s: %w", credentials, err)
		}

		serviceClient, err := serviceFunc(providerClient, gophercloud.EndpointOpts{
			Region: namedCreds.Region,
		})

		if err != nil {
			return fmt.Errorf("unable to create client for %s service with credentials %s: %w", serviceName, credentials, err)
		}

		clientScope := openstackclients.ClientScope{
			NamedCredentials: credentials,
			Project:          namedCreds.Project,
			Domain:           namedCreds.Domain,
			Region:           namedCreds.Region,
		}

		client := openstackclients.Client[*gophercloud.ServiceClient]{
			ClientScope: clientScope,
			Client:      serviceClient,
		}

		clientset.Overwrite(
			clientScope,
			client,
		)

		slog.Info(
			"configured OpenStack client",
			"service", serviceName,
			"credentials", credentials,
			"region", namedCreds.Region,
			"domain", namedCreds.Domain,
			"project", namedCreds.Project,
			"auth_endpoint", namedCreds.AuthEndpoint,
			"auth_method", namedCreds.Authentication,
		)
	}

	return nil
}

// configureOpenStackComputeClientsets configures the OpenStack Compute API clientsets.
func configureOpenStackComputeClientsets(ctx context.Context, conf *config.Config) error {
	return configureOpenStackServiceClientset(ctx, "compute", openstackclients.ComputeClientset, conf.OpenStack.Services.Compute, conf, openstack.NewComputeV2)
}

// configureOpenStackNetworkClientsets configures the OpenStack Network API clientsets.
func configureOpenStackNetworkClientsets(ctx context.Context, conf *config.Config) error {
	return configureOpenStackServiceClientset(ctx, "network", openstackclients.NetworkClientset, conf.OpenStack.Services.Network, conf, openstack.NewNetworkV2)
}

// configureOpenStackObjectStorageClientsets configures the OpenStack Object Storage API clientsets.
func configureOpenStackObjectStorageClientsets(ctx context.Context, conf *config.Config) error {
	return configureOpenStackServiceClientset(ctx, "object_storage", openstackclients.ObjectStorageClientset,
		conf.OpenStack.Services.ObjectStorage, conf, openstack.NewObjectStorageV1)
}

// configureOpenStackLoadBalancerClientsets configures the OpenStack LoadBalancer API clientsets.
func configureOpenStackLoadBalancerClientsets(ctx context.Context, conf *config.Config) error {
	return configureOpenStackServiceClientset(ctx, "load_balancer", openstackclients.LoadBalancerClientset,
		conf.OpenStack.Services.LoadBalancer, conf, openstack.NewLoadBalancerV2)
}

// configureOpenStackIdentityClientsets configures the OpenStack Identity API clientsets.
func configureOpenStackIdentityClientsets(ctx context.Context, conf *config.Config) error {
	return configureOpenStackServiceClientset(ctx, "identity", openstackclients.IdentityClientset, conf.OpenStack.Services.Identity, conf, openstack.NewIdentityV3)
}
