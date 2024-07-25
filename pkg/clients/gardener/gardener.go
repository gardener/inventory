// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardener

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerversioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	machineversioned "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	"golang.org/x/oauth2/google"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/core/registry"
)

// ErrClientNotFound is returned when attempting to get a client, which does not
// exist in the registry.
var ErrClientNotFound = errors.New("client not found")

// ErrNoShoots is returned when there are no shoots registered in the
// virtual garden cluster.
var ErrNoShoots = errors.New("no shoots found")

// ErrShootNotFound is an error, which is returned when a given shoot is not
// found in the virtual garden cluster.
var ErrShootNotFound = errors.New("shoot not found")

// ErrSeedIsExcluded is an error, which is returned when attempting to get a
// [*rest.Config] for a seed cluster, which is excluded in the configuration.
var ErrSeedIsExcluded = errors.New("seed is excluded")

const (
	// VIRTUAL_GARDEN is the name of the virtual garden
	VIRTUAL_GARDEN = "virtual-garden"

	// SOIL_GCP is the name of the GKE soil cluster, which requires Workload Identity Federation to perform OAuth2 token exchange
	SOIL_GCP_REGIONAL = "soil-gcp-regional"

	// VIEWERKUBECONFIG_SUBRESOURCE_PATH is the path to the viewerkubeconfig subresource of a shoot
	// All managed seeds are registered as shoots resources in the virtual-garden in garden namespace
	VIEWERKUBECONFIG_SUBRESOURCE_PATH = "/apis/core.gardener.cloud/v1beta1/namespaces/garden/shoots/%s/viewerkubeconfig"

	// EXPIRATION_SECONDS is the expiration time for the viewkubeconfig client certificate in seconds
	EXPIRATION_SECONDS = `{"spec":{"expirationSeconds":86400}}` // 24h
)

// Client provides the Gardener clients.
type Client struct {
	// restConfigs contains the [rest.Config] items for the various
	// contexts.
	restConfigs *registry.Registry[string, *rest.Config]

	// excludedSeeds represents a list of seed cluster names, from which
	// collection will be skipped and no client config is created for.
	excludedSeeds []string

	// soilRegionalHost is the host of the soil-gcp-regional cluster
	soilRegionalHost string
}

// DefaultClient is the default client for interacting with the Gardener APIs.
var DefaultClient = New()

// SetDefaultClient sets the [DefaultClient] to the specified [Client].
func SetDefaultClient(c *Client) {
	DefaultClient = c
}

// Option is a function, which configures the [Client].
type Option func(c *Client)

// New creates a new [Client].
func New(opts ...Option) *Client {
	c := &Client{
		restConfigs: registry.New[string, *rest.Config](),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithRestConfigs is an [Option], which configures the [Client] with the
// specified map of [*rest.Config] items.
func WithRestConfigs(items map[string]*rest.Config) Option {
	opt := func(c *Client) {
		if items == nil {
			return
		}

		for name, config := range items {
			c.restConfigs.Overwrite(name, config)
		}
	}

	return opt
}

// WithExcludedSeeds is an [Option], which configures the [Client] to skip
// collection from the specified seed cluster names.
func WithExcludedSeeds(seeds []string) Option {
	opt := func(c *Client) {
		c.excludedSeeds = seeds
	}

	return opt
}

func WithSoilRegionalHost(host string) Option {
	opt := func(c *Client) {
		c.soilRegionalHost = host
	}

	return opt
}

// VirtualGarden returns a versioned clientset for the Virtual Garden cluster
func (c *Client) VirtualGardenClient() (*gardenerversioned.Clientset, error) {
	config, found := c.restConfigs.Get(VIRTUAL_GARDEN)
	if !found || config == nil {
		return nil, fmt.Errorf("%w: %s", ErrClientNotFound, VIRTUAL_GARDEN)
	}

	client, err := gardenerversioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// VirtualGardenClient returns the Virtual Garden cluster client using the
// [DefaultClient].
func VirtualGardenClient() (*gardenerversioned.Clientset, error) {
	return DefaultClient.VirtualGardenClient()
}

// Shoots returns the list of shoots registered in the Virtual Garden cluster.
func (c *Client) Shoots(ctx context.Context) ([]*v1beta1.Shoot, error) {
	client, err := c.VirtualGardenClient()
	if err != nil {
		return nil, err
	}

	shoots := make([]*v1beta1.Shoot, 0)
	_ = pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Shoots("").List(ctx, metav1.ListOptions{})
		}),
	).EachListItem(ctx, metav1.ListOptions{}, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Shoot)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		shoots = append(shoots, s)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not list shoots: %w", err)
	}

	return shoots, nil
}

// Shoots returns the list of shoots registered in the Virtual Garden cluster
// using the [DefaultClient].
func Shoots(ctx context.Context) ([]*v1beta1.Shoot, error) {
	return DefaultClient.Shoots(ctx)
}

// MCMClient returns a machine versioned clientset for the given seed name
func (c *Client) MCMClient(name string) (*machineversioned.Clientset, error) {
	if slices.Contains(c.excludedSeeds, name) {
		return nil, fmt.Errorf("%w: %s", ErrSeedIsExcluded, name)
	}

	if name == SOIL_GCP_REGIONAL {
		if _, found := c.restConfigs.Get(name); !found {
			return c.fetchSoilGCPRegionalClient()
		}
	}

	// Check to see if there is a rest.Config with such name, and create
	// config for it, if that's the first time we see it.
	config, found := c.restConfigs.Get(name)
	if !found {
		config, err := c.createGardenConfig(name)
		if err != nil {
			return nil, err
		}
		c.restConfigs.Overwrite(name, config)
		return machineversioned.NewForConfig(config)
	}

	// Make sure our config has not expired
	if goneIn60Seconds(config) {
		config, err := c.createGardenConfig(name)
		if err != nil {
			return nil, err
		}
		c.restConfigs.Overwrite(name, config)
		return machineversioned.NewForConfig(config)
	}

	// If we reach this far, we still have a valid config
	return machineversioned.NewForConfig(config)
}

// MCMClient returns a machine versioned clientset for the given seed name using
// the [DefaultClient].
func MCMClient(name string) (*machineversioned.Clientset, error) {
	return DefaultClient.MCMClient(name)
}

// createGardenConfig creates a [*rest.Config] for the given seed name and
// returns it.
func (c *Client) createGardenConfig(name string) (*rest.Config, error) {
	shoots, err := c.Shoots(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list shoots: %w", err)
	}

	if len(shoots) == 0 {
		return nil, ErrNoShoots
	}

	for _, shoot := range shoots {
		if shoot.Name != name {
			continue
		}

		// send a http request to the viewerkubeconfig subresources
		kubeconfigStr, err := c.fetchSeedKubeconfig(name)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch kubeconfig: %w", err)
		}
		// Shall we add more checks?
		if kubeconfigStr == "" {
			return nil, fmt.Errorf("kubeconfig is empty")
		}

		apiConfig, err := clientcmd.Load([]byte(kubeconfigStr))
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		if apiConfig == nil {
			return nil, fmt.Errorf("config is nil")
		}

		clientConfig := clientcmd.NewNonInteractiveClientConfig(*apiConfig, "garden--"+name+"-external",
			&clientcmd.ConfigOverrides{}, nil)

		restConfig, err := clientConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create rest config: %w", err)
		}

		return restConfig, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrShootNotFound, name)
}

// fetchSeedKubeconfig sends a http request to the viewerkubeconfig subresource
// of a shoot
func (c *Client) fetchSeedKubeconfig(name string) (string, error) {
	config, found := c.restConfigs.Get(VIRTUAL_GARDEN)
	if !found || config == nil {
		return "", fmt.Errorf("garden config not found")
	}

	if config.ContentConfig.GroupVersion == nil {
		config.ContentConfig = rest.ContentConfig{
			GroupVersion:         &v1alpha1.SchemeGroupVersion,
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		}
	}

	client, err := rest.RESTClientFor(config)
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

func (c *Client) fetchSoilGCPRegionalClient() (*machineversioned.Clientset, error) {

	// Load the credentials from the JSON configuration file
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/cloud-platform",
		"openid",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials configuration: %w", err)
	}

	// Use the credentials to create a token source
	tokenSource := creds.TokenSource

	// Get an access token
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch soil-gcp-regional token: %w", err)
	}

	config := &rest.Config{
		Host:        c.soilRegionalHost,
		BearerToken: token.AccessToken,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
	return machineversioned.NewForConfig(config)
}
