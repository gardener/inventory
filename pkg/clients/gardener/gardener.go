// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardener

import (
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
	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	gardenerversioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	machineversioned "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/pager"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gardener/constants"
	gcputils "github.com/gardener/inventory/pkg/gcp/utils"
)

// ErrSeedIsExcluded is an error, which is returned when attempting to get a
// [*rest.Config] for a seed cluster, which is excluded in the configuration.
var ErrSeedIsExcluded = errors.New("seed is excluded")

// ErrNoRestConfig is an error, which is returned when an expected [rest.Config]
// was not configured.
var ErrNoRestConfig = errors.New("no rest.Config specified")

// Client represents the API client used to interface with the Gardener APIs.
type Client struct {
	// restConfig represents the [rest.Config] used to create the
	// Gardener API client.
	restConfig *rest.Config

	// gardenerClient is the API client for interfacing with Gardener
	gardenerClient *gardenerversioned.Clientset

	// userAgent is the User-Agent HTTP header, which will be set on newly
	// created API clients.
	userAgent string

	// seedRestConfigs contains the [rest.Config] items for the known
	// managed seed clusters.
	seedRestConfigs *registry.Registry[string, *rest.Config]

	// excludedSeeds represents a list of seed cluster names, from which
	// collection will be skipped and no client config is created for.
	excludedSeeds []string

	// gkeSoilCluster provides the settings for the GKE soil cluster.
	gkeSoilCluster *GKESoilCluster
}

// GKESoilCluster provides information about a GKE soil cluster, which is
// registered as a Gardener Seed cluster.
type GKESoilCluster struct {
	// SeedName is the name of the seed as registered in Gardener
	SeedName string

	// ClusterName is the name of the GKE cluster.
	ClusterName string

	// CredentialsFile specifies the credentials file to use when performing
	// the OAuth2 token exchange in order to access the GKE Regional Soil
	// cluster.
	CredentialsFile string
}

// DefaultClient is the default client for interfacing with the Gardener APIs.
var DefaultClient *Client

// IsDefaultClientSet is a predicate which returns true when the [DefaultClient]
// has been configured, and returns false otherwise.
func IsDefaultClientSet() bool {
	return DefaultClient != nil
}

// SetDefaultClient sets the [DefaultClient] to the specified [Client].
func SetDefaultClient(c *Client) {
	DefaultClient = c
}

// Option is a function, which configures the [Client].
type Option func(c *Client)

// New creates a new [Client].
func New(opts ...Option) (*Client, error) {
	c := &Client{
		seedRestConfigs: registry.New[string, *rest.Config](),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.restConfig == nil {
		return nil, ErrNoRestConfig
	}

	gardenerClient, err := gardenerversioned.NewForConfig(c.restConfig)
	if err != nil {
		return nil, err
	}
	c.gardenerClient = gardenerClient

	return c, nil
}

// WithRestConfig is an [Option], which configures the [Client] with the
// specified [rest.Config].
func WithRestConfig(restConfig *rest.Config) Option {
	opt := func(c *Client) {
		c.restConfig = restConfig
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

// WithGKESoilCluster is an [Option], which configures the [Client] to use the
// given GKE soil cluster.
func WithGKESoilCluster(settings *GKESoilCluster) Option {
	opt := func(c *Client) {
		c.gkeSoilCluster = settings
	}

	return opt
}

// WithUserAgent is an [Option], which configures the [Client] to set the
// User-Agent header to newly created API clients to the given value.
func WithUserAgent(userAgent string) Option {
	opt := func(c *Client) {
		c.userAgent = userAgent
	}

	return opt
}

// GardenClient returns a [gardenerversioned.Clientset] for interfacing with the
// Gardener APIs.
func (c *Client) GardenClient() *gardenerversioned.Clientset {
	return c.gardenerClient
}

// Seeds returns the list of seeds registered in the Garden cluster.
func (c *Client) Seeds(ctx context.Context) ([]*v1beta1.Seed, error) {
	seeds := make([]*v1beta1.Seed, 0)
	err := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return c.gardenerClient.CoreV1beta1().Seeds().List(ctx, opts)
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
		return nil, err
	}

	return seeds, nil
}

// SeedRestConfig returns a [rest.Config] for the given seed cluster name
func (c *Client) SeedRestConfig(ctx context.Context, name string) (*rest.Config, error) {
	if slices.Contains(c.excludedSeeds, name) {
		return nil, fmt.Errorf("%w: %s", ErrSeedIsExcluded, name)
	}

	// During upgrades of the GKE clusters the CA and public IP address may
	// have changed, while the CA is still valid, and for that reason we
	// create a new [rest.Config] from the latest discovered data.
	if name == c.gkeSoilCluster.SeedName {
		return c.getGKESoilClusterRestConfig(ctx)
	}

	// Check if we have a config and it is still valid
	config, found := c.seedRestConfigs.Get(name)
	if found && !goneIn60Seconds(config) {
		return config, nil
	}

	// Config is either not found, or no longer valid, so we create a new one
	kubeConfig, err := c.ViewerKubeconfig(ctx, v1beta1constants.GardenNamespace, name, 1*time.Hour)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	restConfig.UserAgent = c.userAgent
	c.seedRestConfigs.Overwrite(name, restConfig)

	return restConfig, nil
}

// SeedClient returns a [kubernetes.Clientset] for the given seed cluster name.
func (c *Client) SeedClient(ctx context.Context, name string) (*kubernetes.Clientset, error) {
	config, err := c.SeedRestConfig(ctx, name)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// MCMClient returns a [machineversioned.Clientset] for the given seed cluster
// name.
func (c *Client) MCMClient(ctx context.Context, name string) (*machineversioned.Clientset, error) {
	config, err := c.SeedRestConfig(ctx, name)
	if err != nil {
		return nil, err
	}

	return machineversioned.NewForConfig(config)
}

// ViewerKubeconfig generates a new kubeconfig with read-only access for a shoot
// cluster from a given project namespace.
func (c *Client) ViewerKubeconfig(ctx context.Context, projectNamespace string, shootName string, expiration time.Duration) ([]byte, error) {
	expirationSeconds := int64(expiration.Seconds())
	req := &authenticationv1alpha1.ViewerKubeconfigRequest{
		Spec: authenticationv1alpha1.ViewerKubeconfigRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}

	absPath := fmt.Sprintf(
		"/apis/core.gardener.cloud/v1beta1/namespaces/%s/shoots/%s/viewerkubeconfig",
		projectNamespace,
		shootName,
	)

	scheme := runtime.NewScheme()
	if err := authenticationv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	encoder := serializer.NewCodecFactory(scheme).LegacyCodec(authenticationv1alpha1.SchemeGroupVersion)
	body, err := runtime.Encode(encoder, req)
	if err != nil {
		return nil, fmt.Errorf("viewerkubeconfig: %w", err)
	}

	result := c.gardenerClient.RESTClient().
		Post().
		AbsPath(absPath).
		Body(body).
		Do(ctx)

	if result.Error() != nil {
		return nil, fmt.Errorf("viewerkubeconfig: %w", result.Error())
	}

	if err := result.Into(req); err != nil {
		return nil, fmt.Errorf("viewerkubeconfig: %w", err)
	}

	return req.Status.Kubeconfig, nil
}

// getGKESoilClusterRestConfig returns a [rest.Config] which is configured
// against the GKE Regional cluster (soil cluster).
func (c *Client) getGKESoilClusterRestConfig(ctx context.Context) (*rest.Config, error) {
	// Get the GKE cluster from the data already collected by Inventory,
	// which has already discovered the control-plane endpoint and the CA
	// root of trust for us.
	cluster, err := gcputils.GetGKEClusterFromDB(ctx, c.gkeSoilCluster.ClusterName)
	if err != nil {
		// Cluster does not exist
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("GKE cluster not found in db: %s", c.gkeSoilCluster.ClusterName)
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
		CredentialsFile: c.gkeSoilCluster.CredentialsFile,
	}

	creds, err := credentials.DetectDefault(opts)
	if err != nil {
		return nil, err
	}

	token, err := creds.TokenProvider.Token(ctx) // nolint: staticcheck
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
		UserAgent: c.userAgent,
	}

	return config, nil
}

// goneIn60Seconds is a predicate, which returns true when the expiration time
// of the ClientCertificate or BearerToken is approaching, otherwise it returns
// false.
func goneIn60Seconds(config *rest.Config) bool {
	if config == nil {
		return false
	}

	// Check for the presence of client certificate and its expiration
	if config.TLSClientConfig.CertData != nil { // nolint: staticcheck
		return certIsAboutToExpire(config.TLSClientConfig.CertData) // nolint: staticcheck
	}

	// Check for the presence of file containing a client certificate and its expiration
	if config.TLSClientConfig.CertFile != "" { // nolint: staticcheck
		certData, err := os.ReadFile(config.TLSClientConfig.CertFile) // nolint: staticcheck
		if err != nil {
			slog.Error(fmt.Sprintf("failed to load certificate file: %s", err))

			return true
		}

		return certIsAboutToExpire(certData)
	}

	// Check the presence of BearerToken
	if config.BearerToken != "" {
		return tokenIsAboutToExpire(config.BearerToken)
	}

	// Check the presence of file containing a BearerToken
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

func (c *Client) RESTConfig() *rest.Config {
	return c.restConfig
}
