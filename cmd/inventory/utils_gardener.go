// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/version"
)

// errNoGardenerEndpoint is an error which is returned when no API endpoint has
// been specified.
var errNoGardenerEndpoint = errors.New("no API endpoint specified")

// errNoGardenerKubeconfig is an error, which is returned when an expected
// kubeconfig file was not specified.
var errNoGardenerKubeconfig = errors.New("no kubeconfig file specified")

// errNoGardenerTokenFile is an error, which is returned when an expected token
// file was not specified.
var errNoGardenerTokenFile = errors.New("no token file specified")

// validateGardenerConfig validates the Gardener configuration
func validateGardenerConfig(conf *config.Config) error {
	if conf.Gardener.UserAgent == "" {
		conf.Gardener.UserAgent = fmt.Sprintf("gardener-inventory/%s", version.Version)
	}

	if conf.Gardener.Endpoint == "" {
		return fmt.Errorf("gardener: %w", errNoGardenerEndpoint)
	}

	supportedAuthnMethods := []string{
		config.GardenerAuthenticationMethodInCluster,
		config.GardenerAuthenticationMethodToken,
		config.GardenerAuthenticationMethodKubeconfig,
	}

	if conf.Gardener.Authentication == "" {
		return fmt.Errorf("gardener: %w", errNoAuthenticationMethod)
	}

	if !slices.Contains(supportedAuthnMethods, conf.Gardener.Authentication) {
		return fmt.Errorf("gardener: %w: %s", errUnknownAuthenticationMethod, conf.Gardener.Authentication)
	}

	return nil
}

// getGardenerRestConfig creates a [rest.Config] based on the provided
// [config.Config] settings.
func getGardenerRestConfig(conf *config.Config) (*rest.Config, error) {
	switch conf.Gardener.Authentication {
	case config.GardenerAuthenticationMethodInCluster:
		// In-cluster authentication
		return rest.InClusterConfig()
	case config.GardenerAuthenticationMethodKubeconfig:
		// Kubeconfig authentication
		if conf.Gardener.Kubeconfig == "" {
			kubeconfigFromEnv := os.Getenv("KUBECONFIG")
			if kubeconfigFromEnv == "" {
				return nil, errNoGardenerKubeconfig
			}
			slog.Info(
				"Gardener API client configured via KUBECONFIG",
				"kubeconfig", kubeconfigFromEnv,
			)
			conf.Gardener.Kubeconfig = kubeconfigFromEnv
		}
		return clientcmd.BuildConfigFromFlags("", conf.Gardener.Kubeconfig)
	case config.GardenerAuthenticationMethodToken:
		// Token file authentication
		if conf.Gardener.TokenPath == "" {
			return nil, errNoGardenerTokenFile
		}
		restConfig := &rest.Config{
			Host:            conf.Gardener.Endpoint,
			BearerTokenFile: conf.Gardener.TokenPath,
		}

		return restConfig, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnknownAuthenticationMethod, conf.Gardener.Authentication)
	}
}

// configureGardenerClient configures the API client for interfacing with the
// Gardener APIs.
func configureGardenerClient(_ context.Context, conf *config.Config) error {
	if !conf.Gardener.IsEnabled {
		slog.Warn("Gardener is not enabled, will not create API client")
		return nil
	}

	slog.Info(
		"configuring Gardener API client",
		"authentication", conf.Gardener.Authentication,
		"kubeconfig", conf.Gardener.Kubeconfig,
		"token_path", conf.Gardener.TokenPath,
	)

	restConfig, err := getGardenerRestConfig(conf)
	if err != nil {
		return fmt.Errorf("gardener: %w", err)
	}

	restConfig.UserAgent = conf.Gardener.UserAgent

	gkeSoilClusterConf := &gardenerclient.GKESoilCluster{
		SeedName:        conf.Gardener.SoilClusters.GCP,
		ClusterName:     conf.GCP.SoilCluster.ClusterName,
		CredentialsFile: conf.GCP.Credentials[conf.GCP.SoilCluster.UseCredentials].KeyFile.Path,
	}

	gardenerClientOpts := []gardenerclient.Option{
		gardenerclient.WithRestConfig(restConfig),
		gardenerclient.WithExcludedSeeds(conf.Gardener.ExcludedSeeds),
		gardenerclient.WithGKESoilCluster(gkeSoilClusterConf),
		gardenerclient.WithUserAgent(conf.Gardener.UserAgent),
	}

	gardenClient, err := gardenerclient.New(gardenerClientOpts...)
	if err != nil {
		return err
	}
	gardenerclient.SetDefaultClient(gardenClient)
	slog.Info(
		"configured Gardener API client",
		"host", restConfig.Host,
	)

	return nil
}
