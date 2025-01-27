// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/version"
)

// errNoGardenerEndpoint is an error which is returned when no API endpoint has
// been specified.
var errNoGardenerEndpoint = errors.New("no API endpoint specified")

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
		return fmt.Errorf("gardener: %w: uses %s", errUnknownAuthenticationMethod, conf.Gardener.Authentication)
	}

	return nil
}

func newGardenConfigs(conf *config.Config) (map[string]*rest.Config, error) {
	// 1. Check for token according the configuration
	if conf.VirtualGarden.TokenPath != "" {
		return constructGardenConfigWithToken(conf)
	}

	// 2. Check for kubeconfig in the configuration or an env variable
	// Attempt to read the kubeconfig from the configuration file
	configs := make(map[string]*rest.Config)
	kubeconfig := virtualGardenKubeconfig(conf)
	if kubeconfig != "" {
		// Add any additional contexts from the kubeconfig, if present
		apiConfig, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
		for name := range apiConfig.Contexts {
			contextName := fetchContextName(name, conf.VirtualGarden.Environment)
			clientConfig := clientcmd.NewNonInteractiveClientConfig(*apiConfig, name, &clientcmd.ConfigOverrides{}, nil)
			restConfig, err := clientConfig.ClientConfig()
			if err != nil {
				slog.Error("failed to create rest config, skipping", "context", contextName, "err", err)
				continue
			}
			configs[contextName] = restConfig
		}
		if _, found := configs[gardenerclient.VIRTUAL_GARDEN]; !found {
			return nil, fmt.Errorf("no context found for the virtual garden in the kubeconfig")
		}
		return configs, nil
	}

	// If there is no token and the kubeconfig is not set, we are running in a testing environment
	// 3. Check for in-cluster config - for testing purposes
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}
	configs[gardenerclient.VIRTUAL_GARDEN] = inClusterConfig
	return configs, nil

}

func constructGardenConfigWithToken(conf *config.Config) (map[string]*rest.Config, error) {
	// Check if the token file exists
	configs := make(map[string]*rest.Config)
	var (
		f   os.FileInfo
		err error
	)

	if f, err = os.Stat(conf.VirtualGarden.TokenPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("token file not found: %w", err)
	}
	//check the length of the token file
	if f.Size() == 0 {
		return nil, fmt.Errorf("token file is empty")
	}

	// Create a rest.Config for the Virtual Garden
	restConfig := &rest.Config{
		Host:            fmt.Sprintf("https://api.%s.gardener.cloud.sap", conf.VirtualGarden.Environment),
		BearerTokenFile: conf.VirtualGarden.TokenPath,
	}
	configs[gardenerclient.VIRTUAL_GARDEN] = restConfig
	return configs, nil
}

func fetchContextName(name string, prefix string) string {
	if strings.HasPrefix(name, prefix+"-") {
		return strings.TrimPrefix(name, prefix+"-")
	}
	return name
}

func virtualGardenKubeconfig(conf *config.Config) string {
	if conf.VirtualGarden.Kubeconfig != "" {
		return conf.VirtualGarden.Kubeconfig
	}
	return os.Getenv("KUBECONFIG")
}
