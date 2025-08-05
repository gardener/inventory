// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	gophercloudconfig "github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/pagination"

	openstackclients "github.com/gardener/inventory/pkg/clients/openstack"
	vaultclients "github.com/gardener/inventory/pkg/clients/vault"
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

// openstackVaultSecret provides OpenStack credentials, which were read from a
// Vault secret.
type openstackVaultSecret struct {
	// Kind specifies the kind of OpenStack credentials provided by the
	// secret.  It should be [config.OpenstackVaultSecretKindV3Password] for
	// username/password credentials, or
	// [config.OpenstackVaultSecretKindV3ApplicationCredential] for Application
	// Credentials.
	Kind string `json:"kind,omitempty"`

	// Username for v3password auth type.
	Username string `json:"username,omitempty"`
	// Password for v3password auth type.
	Password string `json:"password,omitempty"`

	// ApplicationCredentialID for v3applicationcredential auth type.
	ApplicationCredentialID string `json:"application_credential_id,omitempty"`
	// ApplicationCredentialSecret for v3applicationcredential auth type.
	ApplicationCredentialSecret string `json:"application_credential_secret,omitempty"`
}

// validateOpenStackConfig validates the OpenStack configuration settings.
func validateOpenStackConfig(conf *config.Config) error {
	// Make sure that the services have named credentials configured.
	services := map[string]config.OpenStackServiceCredentials{
		"compute":        conf.OpenStack.Services.Compute,
		"network":        conf.OpenStack.Services.Network,
		"object_storage": conf.OpenStack.Services.ObjectStorage,
		"load_balancer":  conf.OpenStack.Services.LoadBalancer,
		"identity":       conf.OpenStack.Services.Identity,
		"block_storage":  conf.OpenStack.Services.BlockStorage,
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
				return fmt.Errorf("openstack: %w: %s", errNoUsername, name)
			}
			if creds.Password.PasswordFile == "" {
				return fmt.Errorf("openstack: %w: %s", errNoPasswordFile, name)
			}
		case config.OpenStackAuthenticationMethodAppCredentials:
			if creds.AppCredentials.AppCredentialsID == "" {
				return fmt.Errorf("openstack: %w: %s", errNoAppCredentialsID, name)
			}
			if creds.AppCredentials.AppCredentialsSecretFile == "" {
				return fmt.Errorf("openstack: %w: %s", errNoAppCredentialsSecretFile, name)
			}
		case config.OpenStackAuthenticationMethodVaultSecret:
			if creds.VaultSecret.Server == "" {
				return fmt.Errorf("openstack: no vault server specified for %s", name)
			}

			if _, ok := conf.Vault.Servers[creds.VaultSecret.Server]; !ok {
				return fmt.Errorf("openstack: %s refers to unknown vault server %s", name, creds.VaultSecret.Server)
			}

			if creds.VaultSecret.SecretEngine == "" {
				return fmt.Errorf("openstack: no vault secret engine specified for %s", name)
			}
			if creds.VaultSecret.SecretPath == "" {
				return fmt.Errorf("openstack: no vault secret path specified for %s", name)
			}
		default:
			return fmt.Errorf("openstack: %w: %s uses %s", errUnknownAuthenticationMethod, name, creds.Authentication)
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
				return fmt.Errorf("openstack: %w: %s", errNoServiceCredentials, service)
			}

			if _, ok := conf.OpenStack.Credentials[cred]; !ok {
				return fmt.Errorf("openstack: %w: service %s refers to %s", errUnknownNamedCredentials, service, cred)
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
		"block_storage":  configureOpenStackBlockStorageClientsets,
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
		// Username/password authentication method
		username := strings.TrimSpace(creds.Password.Username)
		if username == "" {
			return nil, fmt.Errorf("no username specified for project %s", creds.Project)
		}

		rawPassword, err := os.ReadFile(filepath.Clean(creds.Password.PasswordFile))
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
		// Application Credentials authentication method
		appID := strings.TrimSpace(creds.AppCredentials.AppCredentialsID)
		if appID == "" {
			return nil, fmt.Errorf("no app credentials id specified for project %s", creds.Project)
		}

		rawAppSecret, err := os.ReadFile(filepath.Clean(creds.AppCredentials.AppCredentialsSecretFile))
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
	case config.OpenStackAuthenticationMethodVaultSecret:
		// Credentials from Vault secret
		if creds.VaultSecret.Server == "" || creds.VaultSecret.SecretEngine == "" || creds.VaultSecret.SecretPath == "" {
			return nil, fmt.Errorf("openstack: invalid vault secret configuration for %s", creds.Project)
		}

		// Read and validate secret contents
		vaultClient, ok := vaultclients.Clientset.Get(creds.VaultSecret.Server)
		if !ok {
			return nil, fmt.Errorf("openstack: vault secret refers to unknown vault server %s", creds.VaultSecret.Server)
		}

		vaultSecret, err := vaultClient.KVv2(creds.VaultSecret.SecretEngine).Get(ctx, creds.VaultSecret.SecretPath)
		if err != nil {
			return nil, fmt.Errorf("openstack: cannot read secret %s/%s from vault: %w", creds.VaultSecret.SecretEngine, creds.VaultSecret.SecretPath, err)
		}

		data, err := json.Marshal(vaultSecret.Data)
		if err != nil {
			return nil, fmt.Errorf("openstack: cannot marshal vault secret %s/%s: %w", creds.VaultSecret.SecretEngine, creds.VaultSecret.SecretPath, err)
		}

		var secret openstackVaultSecret
		if err := json.Unmarshal(data, &secret); err != nil {
			return nil, fmt.Errorf("openstack: cannot unmarshal vault secret %s/%s: %w", creds.VaultSecret.SecretEngine, creds.VaultSecret.SecretPath, err)
		}

		switch secret.Kind {
		case config.OpenStackVaultSecretKindV3Password:
			// Username/password authentication
			if secret.Username == "" || secret.Password == "" {
				return nil, fmt.Errorf("openstack: empty username or password for vault secret %s/%s", creds.VaultSecret.SecretEngine, creds.VaultSecret.SecretPath)
			}
			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint: creds.AuthEndpoint,
				DomainName:       creds.Domain,
				TenantName:       creds.Project,
				Username:         secret.Username,
				Password:         secret.Password,
				AllowReauth:      true,
			}
		case config.OpenStackVaultSecretKindV3ApplicationCredential:
			// Application Credentials authentication
			if secret.ApplicationCredentialID == "" || secret.ApplicationCredentialSecret == "" {
				return nil, fmt.Errorf("openstack: empty app id or app secret for vault secret %s/%s", creds.VaultSecret.SecretEngine, creds.VaultSecret.SecretPath)
			}

			authOpts = gophercloud.AuthOptions{
				IdentityEndpoint:            creds.AuthEndpoint,
				ApplicationCredentialID:     secret.ApplicationCredentialID,
				ApplicationCredentialSecret: secret.ApplicationCredentialSecret,
				AllowReauth:                 true,
			}
		default:
			return nil, fmt.Errorf("openstack: invalid vault secret kind for %s/%s: %q", creds.VaultSecret.SecretEngine, creds.VaultSecret.SecretPath, secret.Kind)
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
		namedCreds, ok := conf.OpenStack.Credentials[credentials]
		if !ok {
			return fmt.Errorf("openstack: %w: %q", errUnknownNamedCredentials, credentials)
		}

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

		projectID, err := getProjectIDForClient(ctx, providerClient, clientScope)
		if err != nil {
			return fmt.Errorf("unable to retrieve project ID: %w", err)
		}

		clientScope.ProjectID = projectID

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

// configureOpenStackBlockStorageClientsets configures the OpenStack BlockStorage API clientsets.
func configureOpenStackBlockStorageClientsets(ctx context.Context, conf *config.Config) error {
	return configureOpenStackServiceClientset(ctx, "block_storage", openstackclients.BlockStorageClientset,
		conf.OpenStack.Services.BlockStorage, conf, openstack.NewBlockStorageV3)
}

func getProjectIDForClient(ctx context.Context, providerClient *gophercloud.ProviderClient, clientScope openstackclients.ClientScope) (string, error) {
	identityClient, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{
		Region: clientScope.Region,
	})
	if err != nil {
		return "", fmt.Errorf("could not create identity client for project metadata: %w", err)
	}

	var projectID string
	found := false

	err = projects.ListAvailable(identityClient).
		EachPage(ctx,
			func(_ context.Context, page pagination.Page) (bool, error) {
				projectList, err := projects.ExtractProjects(page)

				if err != nil {
					return false, fmt.Errorf(
						"could not extract project pages: %w",
						err,
					)
				}

				for _, p := range projectList {
					if p.Name == clientScope.Project {
						projectID = p.ID
						found = true

						break
					}
				}

				return true, nil
			})
	if err != nil {
		return "", fmt.Errorf("could not extract project ID: %w", err)
	}

	if !found {
		return "", fmt.Errorf("project not found: %s", clientScope.Project)
	}

	if projectID == "" {
		return "", fmt.Errorf("project ID is empty")
	}

	return projectID, nil
}
