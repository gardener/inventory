// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardener

import (
	"bytes"
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"cloud.google.com/go/auth/credentials"
	"github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerversioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	machineversioned "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gardener/constants"
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
)

// ErrClientNotFound is returned when attempting to get a client, which does not
// exist in the registry.
var ErrClientNotFound = errors.New("client not found")

// ErrNoSeeds is returned when there are no seeds registered in the virtual
// garden cluster.
var ErrNoSeeds = errors.New("no seeds found")

// ErrSeedNotFound is an error, which is returned when a given seed is not found
// in the virtual garden cluster.
var ErrSeedNotFound = errors.New("seed not found")

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

	// soilClusterName specifies the name of the GKE Regional Soil cluster.
	soilClusterName string

	// soilCredentialsFile specifies the credentials file to use when
	// performing the OAuth2 token exchange in order to access the GKE
	// Regional Soil cluster.
	soilCredentialsFile string
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

// WithSoilClusterName is an [Option], which configures the [Client] to use the
// given GKE cluster name for the Regional Soil cluster.
func WithSoilClusterName(name string) Option {
	opt := func(c *Client) {
		c.soilClusterName = name
	}

	return opt
}

// WithSoilCredentialsFile is an [Option], which configures the [Client] to use
// the given credentials file when performing the OAuth2 token exchange in order
// to access the GKE Regional Soil cluster.
func WithSoilCredentialsFile(path string) Option {
	opt := func(c *Client) {
		c.soilCredentialsFile = path
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

// Seeds returns the list of seeds registered in the Virtual Garden cluster.
func (c *Client) Seeds(ctx context.Context) ([]*v1beta1.Seed, error) {
	client, err := c.VirtualGardenClient()
	if err != nil {
		return nil, err
	}

	seeds := make([]*v1beta1.Seed, 0)
	err = pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.CoreV1beta1().Seeds().List(ctx, opts)
		}),
	).EachListItem(ctx, metav1.ListOptions{Limit: constants.PageSize}, func(obj runtime.Object) error {
		s, ok := obj.(*v1beta1.Seed)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		seeds = append(seeds, s)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("could not list seeds: %w", err)
	}

	return seeds, nil
}

// Seeds returns the list of seeds registered in the Virtual Garden cluster
// using the [DefaultClient].
func Seeds(ctx context.Context) ([]*v1beta1.Seed, error) {
	return DefaultClient.Seeds(ctx)
}

// MCMClient returns a machine versioned clientset for the given seed name
func (c *Client) MCMClient(ctx context.Context, name string) (*machineversioned.Clientset, error) {
	if slices.Contains(c.excludedSeeds, name) {
		return nil, fmt.Errorf("%w: %s", ErrSeedIsExcluded, name)
	}

	if name == SOIL_GCP_REGIONAL {
		return c.fetchSoilGCPRegionalClient(ctx)
	}

	// Check to see if there is a rest.Config with such name, and create
	// config for it, if that's the first time we see it.
	config, found := c.restConfigs.Get(name)
	if !found {
		config, err := c.createGardenConfig(ctx, name)
		if err != nil {
			return nil, err
		}
		c.restConfigs.Overwrite(name, config)
		return machineversioned.NewForConfig(config)
	}

	// Make sure our config has not expired
	if goneIn60Seconds(config) {
		config, err := c.createGardenConfig(ctx, name)
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
func MCMClient(ctx context.Context, name string) (*machineversioned.Clientset, error) {
	return DefaultClient.MCMClient(ctx, name)
}

// createGardenConfig creates a [*rest.Config] for the given seed name and
// returns it.
func (c *Client) createGardenConfig(ctx context.Context, name string) (*rest.Config, error) {
	seeds, err := c.Seeds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list seeds: %w", err)
	}

	if len(seeds) == 0 {
		return nil, ErrNoSeeds
	}

	for _, seed := range seeds {
		if seed.Name != name {
			continue
		}

		// send a http request to the viewerkubeconfig subresources
		kubeconfigStr, err := c.fetchSeedKubeconfig(ctx, name)
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

	return nil, fmt.Errorf("%w: %s", ErrSeedNotFound, name)
}

// fetchSeedKubeconfig sends a http request to the viewerkubeconfig subresource
// of a managed seed.
func (c *Client) fetchSeedKubeconfig(ctx context.Context, name string) (string, error) {
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

// fetchSoilGCPRegionalClient returns a [machineversioned.Clientset] which is
// configured against the GKE Regional cluster (soil cluster).
func (c *Client) fetchSoilGCPRegionalClient(ctx context.Context) (*machineversioned.Clientset, error) {
	// Get the GKE cluster from the data already collected by Inventory,
	// which has already discovered the control-plane endpoint and the CA
	// root of trust for us.
	cluster, err := gcputils.GetGKEClusterFromDB(ctx, c.soilClusterName)
	if err != nil {
		// Cluster does not exist
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("GKE cluster not found in db: %s", c.soilClusterName)
		}
		// Something else occurred
		return nil, err
	}

	// Perform OAuth2 token exchange and use the token to access the GKE
	// cluster.
	opts := &credentials.DetectOptions{
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/cloud-platform",
			"openid",
		},
		CredentialsFile: c.soilCredentialsFile,
	}

	creds, err := credentials.DetectDefault(opts)
	if err != nil {
		return nil, err
	}

	token, err := creds.TokenProvider.Token(ctx)
	if err != nil {
		return nil, err
	}

	caData, err := base64.StdEncoding.DecodeString(cluster.CAData)
	if err != nil {
		return nil, err
	}

	config := &rest.Config{
		Host:        fmt.Sprintf("https://%s", cluster.Endpoint),
		BearerToken: token.Value,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CAData:   caData,
		},
	}

	return machineversioned.NewForConfig(config)
}
