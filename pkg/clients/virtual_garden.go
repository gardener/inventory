// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	gardenerversioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	machineversioned "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gardener/inventory/pkg/core/registry"
)

const (
	// VIRTUAL_GARDEN is the name of the virtual garden
	VIRTUAL_GARDEN = "virtual-garden"

	// VIEWERKUBECONFIG_SUBRESOURCE_PATH is the path to the viewerkubeconfig subresource of a shoot
	// All managed seeds are registered as shoots resources in the virtual-garden in garden namespace
	VIEWERKUBECONFIG_SUBRESOURCE_PATH = "/apis/core.gardener.cloud/v1beta1/namespaces/garden/shoots/%s/viewerkubeconfig"

	// EXPIRATION_SECONDS is the expiration time for the viewkubeconfig client certificate in seconds
	EXPIRATION_SECONDS = `{"spec":{"expirationSeconds":86400}}` // 24h
)

var gardenConfigs = registry.New[string, *rest.Config]()

// VirtualGardenClient returns a gardener versioned clientset for the virtual garden cluster
func VirtualGardenClient() *gardenerversioned.Clientset {
	config, found := gardenConfigs.Get(VIRTUAL_GARDEN)
	if !found || config == nil {
		slog.Error("VirtualGardenClient not found", slog.String("name", VIRTUAL_GARDEN))
		return nil
	}
	client, err := gardenerversioned.NewForConfig(config)
	if err != nil {
		slog.Error("Failed to create VirtualGardenClient", "error", err)
		return nil
	}
	return client
}

// SetGardenConfigs adds the rest.Configs to the gardenConfigs map, overwriting the existing ones
func SetGardenConfigs(clients map[string]*rest.Config) {

	if clients == nil {
		return
	}

	for name, config := range clients {
		gardenConfigs.Unregister(name)
		if err := gardenConfigs.Register(name, config); err != nil {
			slog.Error("Failed to set GardenConfigs", "error", err)
		}
	}
}

// SeedClient returns a machine versioned clientset for the given seed
func SeedClient(name string) *machineversioned.Clientset {

	log := slog.With("name", name)
	// check to see if there is a rest.Config with such name
	_, found := gardenConfigs.Get(name)
	if !found {
		log.Info("SeedClient not found, creating ...")
		if err := createGardenConfig(name); err != nil {
			log.Error("SeedClient not found", "error", err)
			return nil
		}
	}

	config, _ := gardenConfigs.Get(name)
	if goneIn60Seconds(config) {
		log.Info("auth is to expire in 60 seconds, refreshing ...")
		if err := createGardenConfig(name); err != nil {
			log.Error("SeedClient not found", "error", err)
			return nil
		}
	}

	client, err := machineversioned.NewForConfig(config)
	if err != nil {
		log.Error("Failed to create SeedClient", "error", err)
		return nil
	}

	return client
}

// createGardenConfig creates a rest.Config for the given seed name and adds it to the gardenConfigs map
func createGardenConfig(name string) error {
	var (
		c   *gardenerversioned.Clientset
		err error
	)

	gardenConfig, found := gardenConfigs.Get(VIRTUAL_GARDEN)
	if !found || gardenConfig == nil {
		return fmt.Errorf("garden config not found")
	}

	if c, err = gardenerversioned.NewForConfig(gardenConfig); err != nil {
		return fmt.Errorf("failed to create VirtualGardenClient: %w", err)
	}
	shoots, err := c.CoreV1beta1().Shoots("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list shoots: %w", err)
	}
	if shoots == nil {
		return fmt.Errorf("no shoots found")
	}
	for _, shoot := range shoots.Items {
		if shoot.Name != name {
			continue
		}
		// send a http request to the viewerkubeconfig subresources
		kubeconfigStr, err := fetchSeedKubeconfig(name)
		if err != nil {
			return fmt.Errorf("failed to fetch kubeconfig: %w", err)
		}
		// Shall we add more checks?
		if kubeconfigStr == "" {
			return fmt.Errorf("kubeconfig is empty")
		}

		apiConfig, err := clientcmd.Load([]byte(kubeconfigStr))
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if apiConfig == nil {
			return fmt.Errorf("config is nil")
		}

		clientConfig := clientcmd.NewNonInteractiveClientConfig(*apiConfig, "garden--"+name+"-external",
			&clientcmd.ConfigOverrides{}, nil)
		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			return fmt.Errorf("failed to create rest config: %w", err)
		}
		gardenConfigs.Unregister(name)
		if err = gardenConfigs.Register(name, restConfig); err != nil {
			return fmt.Errorf("failed to register garden config: %w", err)
		}

		slog.Info("SeedClient created", slog.String("name", name))
		return nil
	}
	return fmt.Errorf("shoot not found")
}

// fetchSeedKubeconfig sends a http request to the viewerkubeconfig subresource of a shoot
func fetchSeedKubeconfig(name string) (string, error) {

	gardenConfig, found := gardenConfigs.Get(VIRTUAL_GARDEN)
	if !found || gardenConfig == nil {
		return "", fmt.Errorf("garden config not found")
	}

	if gardenConfig.ContentConfig.GroupVersion == nil {
		gardenConfig.ContentConfig = rest.ContentConfig{
			GroupVersion:         &v1alpha1.SchemeGroupVersion,
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		}
	}

	client, err := rest.RESTClientFor(gardenConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create rest client: %w", err)
	}

	// Prepare the path to the viewerkubeconfig subresource
	path := fmt.Sprintf(VIEWERKUBECONFIG_SUBRESOURCE_PATH, name)
	body := bytes.NewBufferString(EXPIRATION_SECONDS) //one week

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result := client.Post().
		AbsPath(path).
		SetHeader("Accept", "application/json").
		Body(body).
		Do(ctx)
	if result.Error() != nil {
		return "", fmt.Errorf("failed to send viewerkubeconfigrequest: %w, path:%s", result.Error(), path)
	}
	request := &v1alpha1.ViewerKubeconfigRequest{}
	if err = result.Into(request); err != nil {
		return "", fmt.Errorf("failed to unmarshal viewerkubeconfigrequest response: %w", err)
	}
	return string(request.Status.Kubeconfig), nil
}

// goneIn60Seconds checks the expiration time of the ClientCertificate or the BearerToken if present
// otherwise returns false
func goneIn60Seconds(config *rest.Config) bool {

	if config == nil {
		return false
	}

	//check for the presence of client certificate and its expiration
	if config.TLSClientConfig.CertData != nil {
		return certIsAboutToExpire(config.TLSClientConfig.CertData)
	}

	//check for the presence of file containing a client certificate and its expiration
	if config.TLSClientConfig.CertFile != "" {
		certData, err := os.ReadFile(config.TLSClientConfig.CertFile)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to load certificate file: %s", err))
			return true
		}
		return certIsAboutToExpire(certData)
	}

	//check the presence of BearerToken
	if config.BearerToken != "" {
		return tokenIsAboutToExpire(config.BearerToken)
	}

	//check the presence of file containing a BearerToken
	if config.BearerTokenFile != "" {
		tokenData, err := os.ReadFile(config.BearerTokenFile)
		if err != nil {
			slog.Error(fmt.Sprintf("failed to load token file: %s", err))
			return true
		}
		return tokenIsAboutToExpire(string(tokenData))
	}

	return false
}

// certIsAboutToExpire checks if the certificate is about to expire
func certIsAboutToExpire(certData []byte) bool {
	b, _ := pem.Decode(certData)
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to parse certificate: %s", err))
		return true
	}
	return (time.Now().UTC().Unix() + 60) > c.NotAfter.UTC().Unix() // gone in 60 seconds
}

// tokenIsAboutToExpire checks if the token is about to expire
func tokenIsAboutToExpire(token string) bool {
	splitToken := strings.Split(token, ".")
	if len(splitToken) != 3 {
		slog.Error("invalid token format")
		return true
	}
	payload, err := base64.RawURLEncoding.DecodeString(splitToken[1])
	if err != nil {
		slog.Error("failed to decode token payload", "error", err)
		return true
	}

	var tokenPayload struct {
		Iss string `json:"iss"`
		Exp int64  `json:"exp"`
	}
	if err = json.Unmarshal(payload, &tokenPayload); err != nil {
		slog.Error("failed to unmarshal token payload", "error", err)
		return true
	}

	return (time.Now().UTC().Unix() + 60) > tokenPayload.Exp // gone in 60 seconds
}
