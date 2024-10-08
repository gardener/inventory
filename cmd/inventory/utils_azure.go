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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"

	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// errAzureNoClientID is an error, which is returned when Azure Workload
// Identity Federation is configured without a client id.
var errAzureNoClientID = errors.New("no client id specified")

// errAzureNoTenantID is an error, which is returned when Azure Workload
// Identity Federation is configured without a tenant id.
var errAzureNoTenantID = errors.New("no tenant id specified")

// errAzureNoTokenFile is an error, which is returned when Azure Workload
// Identity Federation is configured without a token file path.
var errAzureNoTokenFile = errors.New("no token file specified")

// validateAzureConfig validates the Azure configuration settings.
func validateAzureConfig(conf *config.Config) error {
	// Make sure that the services have named credentials configured.
	services := map[string][]string{
		"compute":          conf.Azure.Services.Compute.UseCredentials,
		"resource_manager": conf.Azure.Services.ResourceManager.UseCredentials,
	}

	for service, namedCredentials := range services {
		// We expect named credentials to be specified explicitly
		if len(namedCredentials) == 0 {
			return fmt.Errorf("azure: %w: %s", errNoServiceCredentials, service)
		}

		// Validate that the named credentials are actually defined.
		for _, nc := range namedCredentials {
			if _, ok := conf.Azure.Credentials[nc]; !ok {
				return fmt.Errorf("azure: %w: service %s refers to %s", errUnknownNamedCredentials, service, nc)
			}
		}
	}

	// Validate the named credentials for using valid authentication
	// methods.
	supportedAuthnMethods := []string{
		config.AzureAuthenticationMethodDefault,
		config.AzureAuthenticationMethodWorkloadIdentity,
	}

	for name, creds := range conf.Azure.Credentials {
		if creds.Authentication == "" {
			return fmt.Errorf("azure: %w: credentials %s", errNoAuthenticationMethod, name)
		}
		if !slices.Contains(supportedAuthnMethods, creds.Authentication) {
			return fmt.Errorf("azure: %w: %s uses %s", errUnknownAuthenticationMethod, name, creds.Authentication)
		}
	}

	return nil
}

// configureAzureClients creates the Azure API clients from the specified
// configuration.
func configureAzureClients(ctx context.Context, conf *config.Config) error {
	if !conf.Azure.IsEnabled {
		slog.Warn("Azure is not enabled, will not create API clients")
		return nil
	}

	slog.Info("configuring Azure clients")
	configFuncs := map[string]func(ctx context.Context, conf *config.Config) error{
		"compute":          configureAzureComputeClientsets,
		"resource_manager": configureAzureResourceManagerClientsets,
	}

	for svc, configFunc := range configFuncs {
		if err := configFunc(ctx, conf); err != nil {
			return fmt.Errorf("unable to configure Azure clients for %s: %w", svc, err)
		}
	}

	return nil
}

// getAzureTokenProvider returns an [azcore.TokenCredential] for the given named
// credentials.
func getAzureTokenProvider(conf *config.Config, namedCredentials string) (azcore.TokenCredential, error) {
	creds, ok := conf.Azure.Credentials[namedCredentials]
	if !ok {
		return nil, fmt.Errorf("azure: %w: %s", errUnknownNamedCredentials, namedCredentials)
	}

	switch creds.Authentication {
	case config.AzureAuthenticationMethodDefault:
		return azidentity.NewDefaultAzureCredential(&azidentity.DefaultAzureCredentialOptions{})
	case config.AzureAuthenticationMethodWorkloadIdentity:
		if creds.WorkloadIdentity.ClientID == "" {
			return nil, fmt.Errorf("%w for %s", errAzureNoClientID, namedCredentials)
		}
		if creds.WorkloadIdentity.TenantID == "" {
			return nil, fmt.Errorf("%w for %s", errAzureNoTenantID, namedCredentials)
		}
		if creds.WorkloadIdentity.TokenFile == "" {
			return nil, fmt.Errorf("%w for %s", errAzureNoTokenFile, namedCredentials)
		}

		opts := &azidentity.WorkloadIdentityCredentialOptions{
			ClientID:      creds.WorkloadIdentity.ClientID,
			TenantID:      creds.WorkloadIdentity.TenantID,
			TokenFilePath: creds.WorkloadIdentity.TokenFile,
		}
		return azidentity.NewWorkloadIdentityCredential(opts)
	default:
		return nil, fmt.Errorf("azure: %w: %s", errUnknownAuthenticationMethod, creds.Authentication)
	}
}

// getAzureSubscriptions returns the slice of [armsubscription.Subscription] to
// which the given [azcore.TokenCredential] has access to.
func getAzureSubscriptions(ctx context.Context, creds azcore.TokenCredential) ([]*armsubscription.Subscription, error) {
	factory, err := armsubscription.NewClientFactory(creds, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	client := factory.NewSubscriptionsClient()
	pager := client.NewListPager(&armsubscription.SubscriptionsClientListOptions{})
	result := make([]*armsubscription.Subscription, 0)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}

	return result, nil
}

// configureAzureComputeClientsets configures the Azure Compute API clientsets.
func configureAzureComputeClientsets(ctx context.Context, conf *config.Config) error {
	// For each configured named credential we will get the token provider,
	// then get the list of Subscriptions to which the credentials have
	// access to. Each Subscription is then registered as a client using the
	// respective token provider.
	for _, namedCreds := range conf.Azure.Services.Compute.UseCredentials {
		tokenProvider, err := getAzureTokenProvider(conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the subscriptions to which the current credentials have
		// access to and register each subscription as a known client in
		// our clientset.
		subscriptions, err := getAzureSubscriptions(ctx, tokenProvider)
		if err != nil {
			return err
		}

		for _, subscription := range subscriptions {
			subscriptionID := ptr.Value(subscription.SubscriptionID, "")
			subscriptionName := ptr.Value(subscription.DisplayName, "")
			if subscriptionID == "" {
				return fmt.Errorf("empty subscription id for named credentials %s", namedCreds)
			}

			factory, err := armcompute.NewClientFactory(
				subscriptionID,
				tokenProvider,
				&arm.ClientOptions{},
			)
			if err != nil {
				return err
			}

			// Register Virtual Machines client
			vmClient := factory.NewVirtualMachinesClient()
			azureclients.VirtualMachinesClientset.Overwrite(
				subscriptionID,
				&azureclients.Client[*armcompute.VirtualMachinesClient]{
					NamedCredentials: namedCreds,
					SubscriptionID:   subscriptionID,
					SubscriptionName: subscriptionName,
					Client:           vmClient,
				},
			)
			slog.Info(
				"configured Azure client",
				"service", "compute",
				"sub_service", "virtual-machines",
				"credentials", namedCreds,
				"subscription_id", subscriptionID,
				"subscription_name", subscriptionName,
			)
		}
	}

	return nil
}

// configureAzureResourceManagerClientsets configures the Azure Resource Manager
// API clientsets.
func configureAzureResourceManagerClientsets(ctx context.Context, conf *config.Config) error {
	// Similar to the way we do it for Compute API clients, we first need to
	// get the token provider, and then for each Subscription to which the
	// named credentials have access we create and register an API client.
	for _, namedCreds := range conf.Azure.Services.ResourceManager.UseCredentials {
		tokenProvider, err := getAzureTokenProvider(conf, namedCreds)
		if err != nil {
			return err
		}

		// Get the subscriptions to which the current credentials have
		// access to and register each subscription as a known client in
		// our clientset.
		subscriptions, err := getAzureSubscriptions(ctx, tokenProvider)
		if err != nil {
			return err
		}

		for _, subscription := range subscriptions {
			subscriptionID := ptr.Value(subscription.SubscriptionID, "")
			subscriptionName := ptr.Value(subscription.DisplayName, "")
			if subscriptionID == "" {
				return fmt.Errorf("empty subscription id for named credentials %s", namedCreds)
			}

			subFactory, err := armsubscription.NewClientFactory(tokenProvider, &arm.ClientOptions{})
			if err != nil {
				return err
			}

			// Register Subscription clients
			subClient := subFactory.NewSubscriptionsClient()
			azureclients.SubscriptionsClientset.Overwrite(
				subscriptionID,
				&azureclients.Client[*armsubscription.SubscriptionsClient]{
					NamedCredentials: namedCreds,
					SubscriptionID:   subscriptionID,
					SubscriptionName: subscriptionName,
					Client:           subClient,
				},
			)
			slog.Info(
				"configured Azure client",
				"service", "resource_manager",
				"sub_service", "subscriptions",
				"credentials", namedCreds,
				"subscription_id", subscriptionID,
				"subscription_name", subscriptionName,
			)

			// Register Resource Groups clients
			rgFactory, err := armresources.NewClientFactory(
				subscriptionID,
				tokenProvider,
				&arm.ClientOptions{},
			)
			if err != nil {
				return err
			}

			rgClient := rgFactory.NewResourceGroupsClient()
			azureclients.ResourceGroupsClientset.Overwrite(
				subscriptionID,
				&azureclients.Client[*armresources.ResourceGroupsClient]{
					NamedCredentials: namedCreds,
					SubscriptionID:   subscriptionID,
					SubscriptionName: subscriptionName,
					Client:           rgClient,
				},
			)
			slog.Info(
				"configured Azure client",
				"service", "resource_manager",
				"sub_service", "resource-groups",
				"credentials", namedCreds,
				"subscription_id", subscriptionID,
				"subscription_name", subscriptionName,
			)
		}
	}

	return nil
}
