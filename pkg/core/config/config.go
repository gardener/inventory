// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
)

const (
	// DefaultAWSTokenRetriever is the name of the default AWS Token
	// Retriever.
	DefaultAWSTokenRetriever = "none"

	// DefaultAWSAppID is the name of the default AWS App ID.
	DefaultAWSAppID = "gardener-inventory"

	// GCPAuthenticationMethodNone is the name of the default authentication
	// method/strategy to use when creating GCP API clients.  In this
	// strategy Application Default Credentials (ADC) is used when
	// configuring the API clients.
	GCPAuthenticationMethodNone = "none"

	// GCPAuthenticationMethodKeyFile is the name of the authentication
	// method/strategy to use when creating API clients, which are
	// authenticated using service account JSON key files.
	GCPAuthenticationMethodKeyFile = "key_file"

	// AzureAuthenticationMethodDefault is the name of the authentication
	// mechanism for Azure, which uses the [DefaultAzureCredential] chain of
	// credential providers.
	//
	// [DefaultAzureCredential]: https://learn.microsoft.com/en-us/azure/developer/go/sdk/authentication-overview
	AzureAuthenticationMethodDefault = "default"

	// AzureAuthenticationMethodWorkloadIdentity is the name of the
	// authentication mechanism for Azure, which uses [Workload Identity Federation].
	//
	// [Workload Identity Federation]: https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation
	AzureAuthenticationMethodWorkloadIdentity = "workload_identity"

	// GardenerAuthenticationMethodInCluster is the name of the method for
	// `in_cluster' authentication.
	GardenerAuthenticationMethodInCluster = "in_cluster"

	// GardenerAuthenticationMethodToken is the name of the method for
	// `token' authentication.
	GardenerAuthenticationMethodToken = "token"

	// GardenerAuthenticationMethodKubeconfig is the name of the method for
	// `kubeconfig' authentication.
	GardenerAuthenticationMethodKubeconfig = "kubeconfig"

	// DefaultQueueName is the name of the queue which will be used by the
	// client, scheduler and workers, when no queue has been specified
	// explicitly.
	DefaultQueueName = "default"

	// OpenStackAuthenticationMethodPassword is the name of the
	// authentication mechanism for OpenStack, which uses username/password.
	OpenStackAuthenticationMethodPassword = "password"

	// OpenStackAuthenticationMethodAppCredentials is the name of the
	// authentication mechanism for OpenStack, which uses application credentials.
	OpenStackAuthenticationMethodAppCredentials = "app_credentials"

	// OpenStackAuthenticationMethodVaultSecret is the name of the
	// authentication mechanism for OpenStack, which reads credentials from
	// a Vault secret.
	OpenStackAuthenticationMethodVaultSecret = "vault_secret"

	// OpenStackVaultSecretKindV3Password is a Vault secret kind for
	// OpenStack credentials using username/password.
	OpenStackVaultSecretKindV3Password = "v3password"

	// OpenStackVaultSecretKindV3ApplicationCredential is a Vault secret kind for
	// OpenStack credentials using Application Credentials.
	OpenStackVaultSecretKindV3ApplicationCredential = "v3applicationcredential"

	// DefaultWorkerMetricsAddress is the network address from which the
	// worker is serving metrics.
	DefaultWorkerMetricsAddress = ":6080"

	// DefaultWorkerMetricsPath is the default HTTP path at which the worker
	// is exposing metrics.
	DefaultWorkerMetricsPath = "/metrics"
)

// ErrNoConfigVersion error is returned when the configuration does not specify
// config format version.
var ErrNoConfigVersion = errors.New("config format version not specified")

// ErrUnsupportedVersion is an error, which is returned when the config file
// uses an incompatible version format.
var ErrUnsupportedVersion = errors.New("unsupported config format version")

// ConfigFormatVersion represents the supported config format version.
const ConfigFormatVersion = "v1alpha1"

// Config represents the Inventory configuration.
type Config struct {
	// Version is the version of the config file.
	Version string `yaml:"version"`

	// Debug configures debug mode, if set to true.
	Debug bool `yaml:"debug"`

	// Logging provides the logging config settings
	Logging LoggingConfig `yaml:"logging"`

	// Redis represents the Redis configuration
	Redis RedisConfig `yaml:"redis"`

	// Database represents the database configuration.
	Database DatabaseConfig `yaml:"database"`

	// Worker represents the worker configuration.
	Worker WorkerConfig `yaml:"worker"`

	// Scheduler represents the scheduler configuration.
	Scheduler SchedulerConfig `yaml:"scheduler"`

	// Gardener represents the Gardener specific configuration.
	Gardener GardenerConfig `yaml:"gardener"`

	// Dashboard represents the configuration for the Dashboard
	// service.
	Dashboard DashboardConfig `yaml:"dashboard"`

	// AWS represents the AWS specific configuration settings.
	AWS AWSConfig `yaml:"aws"`

	// GCP represents the GCP specific configuration settings.
	GCP GCPConfig `yaml:"gcp"`

	// Azure represents the Azure specific configuration settings.
	Azure AzureConfig `yaml:"azure"`

	// OpenStack represents the OpenStack specific configuration settings.
	OpenStack OpenStackConfig `yaml:"openstack"`

	// Vault represents the Vault specific config settings.
	Vault VaultConfig `yaml:"vault"`
}

// VaultConfig provides the Vault-related configuration.
type VaultConfig struct {
	// IsEnabled specifies whether Vault is enabled or not. Setting this to
	// false will not create any Vault API client.
	IsEnabled bool `yaml:"is_enabled"`

	// Servers specifies the Vault servers configuration.
	Servers map[string]VaultEndpointConfig `yaml:"servers"`
}

// VaultEndpointConfig provides the config settings for a Vault server endpoint.
type VaultEndpointConfig struct {
	// Endpoint represents the Vault server URL
	Endpoint string `yaml:"endpoint"`

	// Namespace specifies the Vault namespace to use.
	Namespace string `yaml:"namespace"`

	// TLSConfig specifies the TLS config for this Vault server
	TLSConfig VaultEndpointTLSConfig `yaml:"tls"`

	// AuthMethod specifies the Authentication Method to use when creating
	// API clients. The currently supported Auth Methods are `token' and
	// `jwt'.
	AuthMethod string `yaml:"auth_method"`

	// TokenAuth specifies the auth settings when using `token' auth method.
	TokenAuth VaultTokenAuthMethodConfig `yaml:"token_auth"`

	// JWTAuth specifies the auth settings when using `jwt' auth method.
	JWTAuth VaultJWTAuthMethodConfig `yaml:"jwt_auth"`
}

// VaultEndpointTLSConfig provides the TLS settings for a Vault server endpoint.
//
// This struct is based on [vault.TLSConfig] with included tags for YAML
// decoding.
type VaultEndpointTLSConfig struct {
	// CACert is the path to a PEM-encoded CA cert file to use to verify the
	// Vault server SSL certificate. It takes precedence over CACertBytes
	// and CAPath.
	CACert string `yaml:"ca_cert"`

	// CACertBytes is a PEM-encoded certificate or bundle. It takes precedence
	// over CAPath.
	CACertBytes []byte `yaml:"ca_cert_bytes"`

	// CAPath is the path to a directory of PEM-encoded CA cert files to verify
	// the Vault server SSL certificate.
	CAPath string `yaml:"ca_path"`

	// ClientCert is the path to the certificate for Vault communication
	ClientCert string `yaml:"client_cert"`

	// ClientKey is the path to the private key for Vault communication
	ClientKey string `yaml:"client_key"`

	// TLSServerName, if set, is used to set the SNI host when connecting via
	// TLS.
	TLSServerName string `yaml:"tls_server_name"`

	// Insecure enables or disables SSL verification
	Insecure bool `yaml:"insecure"`
}

// VaultTokenAuthMethodConfig provides the auth settings when using `token' auth
// method in Vault.
type VaultTokenAuthMethodConfig struct {
	// TokenPath specifies a path to a file containing Vault token
	TokenPath string `yaml:"token_path"`
}

// VaultJWTAuthMethodConfig provides the auth settings when using `jwt' auth
// method in Vault.
type VaultJWTAuthMethodConfig struct {
	// MountPath specifies the JWT Auth Method mount path.
	MountPath string `yaml:"mount_path"`

	// RoleName specifies the role to use.
	RoleName string `yaml:"role_name"`

	// TokenPath specifies a path to a file containing the JWT token to be
	// used when logging into the JWT Auth Method endpoint.
	TokenPath string `yaml:"token_path"`

	// TokenEnv specifies the name of an environment variable, from which to
	// read the JWT token.
	TokenEnv string `yaml:"token_env"`
}

// OpenStackConfig provides the OpenStack-related configuration.
type OpenStackConfig struct {
	// IsEnabled specifies whether the OpenStack collection is enabled or not.
	// Setting this to false will not create any OpenStack client.
	IsEnabled bool `yaml:"is_enabled"`

	// Services provides the OpenStack service-specific configuration.
	Services OpenStackServices `yaml:"services"`

	// Credentials specifies the OpenStack named credentials configuration,
	// which is used by the various OpenStack services.
	Credentials map[string]OpenStackCredentialsConfig `yaml:"credentials"`
}

// OpenStackServices repsesents the known OpenStack services and their config.
type OpenStackServices struct {
	// Compute provides the Compute service configuration.
	Compute OpenStackServiceCredentials `yaml:"compute"`

	// Network provides the Network service configuration.
	Network OpenStackServiceCredentials `yaml:"network"`

	// ObjectStorage provides the object Storage service configuration.
	ObjectStorage OpenStackServiceCredentials `yaml:"object_storage"`

	// LoadBalancer provides the LoadBalancer service configuration.
	LoadBalancer OpenStackServiceCredentials `yaml:"load_balancer"`

	// Identity provides the Identity service configuration.
	Identity OpenStackServiceCredentials `yaml:"identity"`

	// BlockStorage provides the BlockStorage service configuration.
	BlockStorage OpenStackServiceCredentials `yaml:"block_storage"`
}

// OpenStackServiceCredentials specifies which credentials a service can use.
type OpenStackServiceCredentials struct {
	// UseCredentials specifies a list of named credentials to use.
	UseCredentials []string `yaml:"use_credentials"`
}

// OpenStackCredentialsConfig provides named credentials configuration for the OpenStack
// API clients.
type OpenStackCredentialsConfig struct {
	// Authentication specifies the authentication method/strategy to use
	// when creating OpenStack API clients. The currently supported
	// authentication mechanisms are `password' for username/password,
	// `app_credentials' for Application Credentials and `vault_secret' for
	// reading credentials from a Vault secret.
	Authentication string `yaml:"authentication"`

	// Password provides the settings to use for authentication when using username/password.
	Password OpenStackPasswordConfig `yaml:"password"`

	// AppCredentials provides the settings to use for authentication when using application credentials.
	AppCredentials OpenStackAppCredentialsConfig `yaml:"app_credentials"`

	// VaultSecret specifies config settings for reading OpenStack
	// credentials from a Vault secret.
	VaultSecret OpenStackVaultSecretConfig `yaml:"vault_secret"`

	// Domain specifies the domain to use when initializing an OpenStack client.
	Domain string `yaml:"domain"`

	// Project specifies the project to use when initializing an OpenStack client.
	Project string `yaml:"project"`

	// Region specifies the region to use when initializing an OpenStack client.
	Region string `yaml:"region"`

	// AuthEndpoint specifies the authentication endpoint to use when initializing an OpenStack client.
	AuthEndpoint string `yaml:"auth_endpoint"`
}

// OpenStackVaultSecretConfig provides the config settings for reading OpenStack
// credentials from a Vault secret.
type OpenStackVaultSecretConfig struct {
	// Server specifies the name of the Vault server, which contains the
	// secret.
	Server string `yaml:"server"`

	// SecretEngine is the mount path for a KV v2 secret engine, which
	// provides the secret.
	SecretEngine string `yaml:"secret_engine"`

	// SecretPath specifies the path to the secret.
	SecretPath string `yaml:"secret_path"`
}

// OpenStackPasswordConfig provides the settings to use for authentication when using username/password.
type OpenStackPasswordConfig struct {
	// Username specifies the username to use.
	Username string `yaml:"username"`

	// PasswordFile specifies the file path of the file containing the password to use.
	PasswordFile string `yaml:"password_file"`
}

// OpenStackAppCredentialsConfig provides the settings to use for authentication when using application credentials.
type OpenStackAppCredentialsConfig struct {
	// AppCredentialsID specifies the application credential ID to use when authenticating.
	AppCredentialsID string `yaml:"app_credentials_id"`

	// AppCredentialsSecretFile specifies the file path containing the application credential secret to use when authenticating.
	AppCredentialsSecretFile string `yaml:"app_credentials_secret_file"`
}

// AzureConfig provides Azure specific configuration settings.
type AzureConfig struct {
	// IsEnabled specifies whether the Azure collection is enabled or not.
	// Setting this to false will not create any Azure API client.
	IsEnabled bool `yaml:"is_enabled"`

	// Services provides the Azure service-specific configuration.
	Services AzureServices `yaml:"services"`

	// Credentials specifies the Azure named credentials configuration,
	// which is used by the various Azure services.
	Credentials map[string]AzureCredentialsConfig `yaml:"credentials"`
}

// AzureServices repsesents the known Azure services and their config.
type AzureServices struct {
	// Compute provides the Compute service configuration.
	Compute AzureServiceConfig `yaml:"compute"`

	// ResourceManager provides the Resource Manager service configuration.
	ResourceManager AzureServiceConfig `yaml:"resource_manager"`

	// Network provides the Network service configuration.
	Network AzureServiceConfig `yaml:"network"`

	// Storage provides the Storage service configuration.
	Storage AzureServiceConfig `yaml:"storage"`

	// Graph provides the Graph API service configuration.
	Graph AzureServiceConfig `yaml:"graph"`
}

// AzureServiceConfig provides configuration specific for an Azure service.
type AzureServiceConfig struct {
	// UseCredentials specifies the name of the credentials to use.
	UseCredentials []string `yaml:"use_credentials"`
}

// AzureCredentialsConfig provides named credentials configuration for the Azure
// API clients.
type AzureCredentialsConfig struct {
	// Authentication specifies the authentication mechanism to use when
	// creating Azure API clients.
	//
	// The currently supported authentication mechanisms are `default' and
	// `workload_identity'.
	//
	// When using `default' as the authentication mechanism the API client
	// will be initialized with the DefaultAzureCredential chain of
	// credential providers [1].
	//
	// When using `workload_identity' as the authentication mechanism, the
	// API client will be configured to authenticate using Workload Identity
	// Federation [2].
	//
	// [1]: https://learn.microsoft.com/en-us/azure/developer/go/sdk/authentication-overview
	// [2]: https://learn.microsoft.com/en-us/entra/workload-id/workload-identity-federation
	Authentication string `yaml:"authentication"`

	// WorkloadIdentity provides the config settings for authentication
	// using Workload Identity Federation.
	WorkloadIdentity AzureWorkloadIdentityConfig `yaml:"workload_identity"`
}

// AzureWorkloadIdentityConfig provides the config settings for Azure Workload
// Identity Federation.
type AzureWorkloadIdentityConfig struct {
	// ClientID specifies the service principal.
	ClientID string `yaml:"client_id"`

	// TenantID specifies the tenant of the service principal.
	TenantID string `yaml:"tenant_id"`

	// TokenFile specifies the path to a file, which contains the JWT token,
	// which will be exchanged for Azure access token.
	TokenFile string `yaml:"token_file"`
}

// GCPConfig provides GCP specific configuration settings.
type GCPConfig struct {
	// IsEnabled specifies whether the GCP collection is enabled or not.
	// Setting this to false will not create any GCP client.
	IsEnabled bool `yaml:"is_enabled"`

	// UserAgent is the User-Agent header to configure for the API clients.
	UserAgent string `yaml:"user_agent"`

	// Services provides the GCP service-specific configuration.
	Services GCPServices `yaml:"services"`

	// Credentials specifies the GCP named credentials configuration, which
	// is used by the various GCP services.
	Credentials map[string]GCPCredentialsConfig `yaml:"credentials"`

	// SoilCluster specifies the configuration settings for the GKE Regional
	// Soil cluster.
	SoilCluster GCPSoilClusterConfig `yaml:"soil_cluster"`
}

// GCPSoilClusterConfig provides config settings specific to the GKE Regional
// Soil cluster.
type GCPSoilClusterConfig struct {
	// ClusterName specifies the name of the GKE cluster.
	ClusterName string `yaml:"cluster_name"`

	// UseCredentials specifies the named credentials to use when creating
	// an API client to communicate with the GCP Regional Soil cluster.
	UseCredentials string `yaml:"use_credentials"`
}

// GCPServices provides service-specific configuration for the GCP services.
type GCPServices struct {
	// ResourceManager contains the Resource Manager service configuration.
	ResourceManager GCPServiceConfig `yaml:"resource_manager"`

	// Compute contains the Compute Service configuration.
	Compute GCPServiceConfig `yaml:"compute"`

	// Storage contains the Storage Service configuration.
	Storage GCPServiceConfig `yaml:"storage"`

	// GKE contains the GKE service configuration.
	GKE GCPServiceConfig `yaml:"gke"`
}

// GCPServiceConfig provides service-specific configuration for a GCP service.
type GCPServiceConfig struct {
	// UseCredentials specifies the name of the credentials to use.
	UseCredentials []string `yaml:"use_credentials"`
}

// GCPCredentialsConfig provides named credentials configuration for the GCP API
// clients.
type GCPCredentialsConfig struct {
	// Authentication specifies the authentication method/strategy to use
	// when creating GCP API clients.
	//
	// The currently supported authentication strategies are `none' and
	// `key_file'.
	//
	// When using `none' as the authentication strategy the GCP API client
	// will be initialized with Application Default Credentials (ADC) [1].
	//
	// When using `key_file' as the authentication strategy, the GCP API
	// client will be configured to authenticate using the specified service
	// account JSON key file [2].
	//
	// [1]: https://cloud.google.com/docs/authentication/application-default-credentials
	// [2]: https://cloud.google.com/iam/docs/keys-create-delete
	Authentication string `yaml:"authentication"`

	// Projects specifies the list of projects the credentials are valid
	// for.  When creating the respective GCP API clients collection will
	// happen only against the specified projects.
	Projects []string `yaml:"projects"`

	// KeyFile provides the settings to use for authentication when using
	// service account JSON Key File [1].
	//
	// [1]: https://cloud.google.com/iam/docs/keys-create-delete
	KeyFile GCPKeyFile `yaml:"key_file"`
}

// GCPKeyFile provides the authentication settings for using service account
// JSON Key File.
type GCPKeyFile struct {
	// Path specifies the path to the service account JSON key file.
	Path string `yaml:"path"`
}

// AWSConfig provides AWS specific configuration settings.
type AWSConfig struct {
	// IsEnabled specifies whether the AWS collection is enabled or not.
	// Setting this to false will not create any AWS client.
	IsEnabled bool `yaml:"is_enabled"`

	// Region is the region to use when initializing the AWS client.
	Region string `yaml:"region"`

	// DefaultRegion is the default region to use when initializing the AWS client.
	DefaultRegion string `yaml:"default_region"`

	// AppID is an optional application specific identifier.
	AppID string `yaml:"app_id"`

	// Services provides AWS service-specific configuration,
	// e.g. credentials to use when accessing a given AWS service.
	Services AWSServices `yaml:"services"`

	// Credentials specifies the AWS credentials configuration, which is
	// used by the various AWS services.
	Credentials map[string]AWSCredentialsConfig `yaml:"credentials"`
}

// AWSServices provides service-specific configuration for the AWS services.
type AWSServices struct {
	// EC2 contains EC2-specific service configuration
	EC2 AWSServiceConfig `yaml:"ec2"`

	// ELB contains ELBv1-specific service configuration
	ELB AWSServiceConfig `yaml:"elb"`

	// ELBv2 contains ELBv2-specific service configuration
	ELBv2 AWSServiceConfig `yaml:"elbv2"`

	// S3 provides S3-specific service configuration
	S3 AWSServiceConfig `yaml:"s3"`

	// Route53 provides Route 53-specific service configuration
	Route53 AWSServiceConfig `yaml:"route53"`
}

// AWSServiceConfig prvides service-specific configuration for an AWS service.
type AWSServiceConfig struct {
	// UseCredentials specifies the name of the credentials to use for a
	// given AWS Service.
	UseCredentials []string `yaml:"use_credentials"`
}

// AWSCredentialsConfig provides credentials specific configuration for the AWS
// client.
type AWSCredentialsConfig struct {
	// TokenRetriever specifies the name of the token retriever to be used.
	//
	// The token retriever, in combination with Web Identity Credentials
	// Provider is used for retrieving JWT identity tokens, which are then
	// exchanged for temporary security credentials when accessing AWS
	// resources.
	//
	// The currently supported token retrievers are: `none', `kube_sa_token'
	// and `token_file'.
	//
	// When using the `none' token retriever the AWS client will be
	// initialized using the shared credentials file at ~/.aws/credentials
	// without creating a Web Identity Credentials Provider.
	//
	// With the `kube_sa_token' retriever the AWS client will be initialized
	// with a Web Identity Credentials provider, which uses Kubernetes
	// service account tokens, which are then exchanged for temporary
	// security credentials when communicating with the AWS services.
	//
	// When using the `token_file' retriever the AWS client will be
	// initialized with a Web Identity Credentials Provider, which will read
	// JWT identity tokens from a specified path. The JWT token will be
	// exchanged for temporary security credentials for AWS, in a way
	// similar to the `kube_sa_token' retriever.
	//
	// When using `kube_sa_token' and `token_file' retrievers it is assumed
	// that OIDC Trust is already established between the OIDC Providers and
	// AWS.
	TokenRetriever string `yaml:"token_retriever"`

	// KubeSATokenRetriever provides the configuration settings for the
	// Kubernetes Service Account Token Retriever.
	KubeSATokenRetriever AWSKubeSATokenRetrieverConfig `yaml:"kube_sa_token"`

	// TokenFileRetriever provides the configuration settings for the Token
	// File retriever.
	TokenFileRetriever AWSTokenFileRetrieverConfig `yaml:"token_file"`
}

// AWSKubeSATokenRetrieverConfig represents the configuration settings for the
// AWS Kubernetes Service Account Token retriever.
type AWSKubeSATokenRetrieverConfig struct {
	// Kubeconfig specifies the path to a Kubeconfig file to use when
	// creating the underlying Kubernetes client. If empty, the Kubernetes
	// client will be created using in-cluster configuration.
	Kubeconfig string `yaml:"kubeconfig"`

	// ServiceAccount specifies the Kubernetes service account name.
	ServiceAccount string `yaml:"service_account"`

	// Namespace specifies the Kubernetes namespace of the service account.
	Namespace string `yaml:"namespace"`

	// Duration specifies the expiry duration for the service account token
	// and STS credentials.
	Duration time.Duration `yaml:"duration"`

	// Audiences specifies the list of audiences the service account token
	// will be issued for.
	Audiences []string `yaml:"audiences"`

	// RoleARN specifies the IAM Role ARN to be assumed.
	RoleARN string `yaml:"role_arn"`

	// RoleSessionName is a unique name for the session.
	RoleSessionName string `yaml:"role_session_name"`
}

// AWSTokenFileRetrieverConfig represents the configuration settings for the AWS
// Token File retriever.
type AWSTokenFileRetrieverConfig struct {
	// Path specifies the path to the identity token file.
	Path string `yaml:"path"`

	// RoleARN specifies the IAM Role ARN to be assumed.
	RoleARN string `yaml:"role_arn"`

	// RoleSessionName is a unique name for the session.
	RoleSessionName string `yaml:"role_session_name"`

	// Duration specifies the expiry duration for the STS credentials.
	Duration time.Duration `yaml:"duration"`
}

// RedisConfig provides Redis specific configuration settings.
type RedisConfig struct {
	// Endpoint is the endpoint of the Redis service.
	Endpoint string `yaml:"endpoint"`
}

// DatabaseConfig provides database specific configuration settings.
type DatabaseConfig struct {
	// DSN is the Data Source Name to connect to.
	DSN string `yaml:"dsn"`

	// MigrationDirectory specifies an alternate location with migration
	// files.
	MigrationDirectory string `yaml:"migration_dir"`
}

// WorkerConfig provides worker specific configuration settings.
type WorkerConfig struct {
	// Metrics specifies the settings for exposing metrics from workers.
	Metrics WorkerMetricsConfig `yaml:"metrics"`

	// Concurrency specifies the concurrency level for workers.
	Concurrency int `yaml:"concurrency"`

	// Queues specifies the priority queue configuration for the worker.
	//
	// See [1] for more details about how priority queues work.
	//
	// [1]: https://github.com/hibiken/asynq/wiki/Queue-Priority
	Queues map[string]int `yaml:"queues"`

	// StrictPriority specifies whether queue priority is treated strictly.
	//
	// When it is set to true tasks from queues with higher priority are
	// always processed first, and tasks from queues with lower priority are
	// processed only after higher priority queues are empty.
	StrictPriority bool `yaml:"strict_priority"`
}

// WorkerMetricsConfig provides settings for exposing worker-related metrics
type WorkerMetricsConfig struct {
	// Path specifies the HTTP path at which metrics will be exposed.
	Path string `yaml:"path"`

	// Address specifies the TCP network address for the HTTP server, which
	// serves the metrics.
	Address string `yaml:"address"`
}

// SchedulerConfig provides scheduler specific configuration settings.
type SchedulerConfig struct {
	// DefaultQueue specifies the queue name to which tasks will be
	// submitted, if a periodic job does not specify a queue explicitly
	DefaultQueue string `yaml:"default_queue"`

	// Jobs represents the periodic jobs managed by the scheduler
	Jobs []*PeriodicJob `yaml:"jobs"`
}

// PeriodicJob is a job, which is enqueued by the scheduler on regular basis and
// is processed by workers.
type PeriodicJob struct {
	// Name specifies the name of the task to be enqueued
	Name string `yaml:"name"`

	// Spec represents the cron spec for the task
	Spec string `yaml:"spec"`

	// Desc is an optional description associated with the job
	Desc string `yaml:"desc"`

	// Payload is an optional payload to use when submitting the task.
	Payload string `yaml:"payload"`

	// Queue specifies the name of the queue to which the task will be
	// submitted. If it is not specified, then the task will be submitted to
	// the [DefaultQueueName] queue.
	Queue string `yaml:"queue"`
}

// GardenerConfig represents the Gardener specific configuration.
type GardenerConfig struct {
	// IsEnabled specifies whether the Gardener collection is enabled or
	// not.  Setting this to false will not configure a Gardener API client.
	IsEnabled bool `yaml:"is_enabled"`

	// UserAgent is the User-Agent header to configure for the API client.
	UserAgent string `yaml:"user_agent"`

	// Endpoint specifies the endpoint of the Gardener APIs.
	Endpoint string `yaml:"endpoint"`

	// Authentication specifies the mechanism for authentication when
	// interfacing with the Gardener APIs. The currently supported
	// authentication mechanisms are `in_cluster', `token' and `kubeconfig'.
	//
	// When using `in_cluster' the API client will be initialized using
	// using Bearer tokens mounted into pods from well-known location.
	//
	// With `token' mechanism the API client will be initialized using a
	// Bearer token provided from a specified path.
	//
	// With `kubeconfig' authentication mechanism the API client will be
	// initialized using a specified kubeconfig file.
	//
	// For more details please refer to [1].
	//
	// [1]: https://kubernetes.io/docs/reference/access-authn-authz/authentication/
	Authentication string `yaml:"authentication"`

	// TokenPath represents a path to a token file, which will be used to
	// authenticate against the Gardener APIs. The token should be signed by
	// an Identity Provider which is trusted by Gardener.
	TokenPath string `yaml:"token_path"`

	// Kubeconfig represents a path to a kubeconfig file, which will be used
	// to authenticate against Gardener APIs.
	Kubeconfig string `yaml:"kubeconfig"`

	// ExcludedSeeds is a list of seed cluster names, from which collection
	// will be skipped.
	ExcludedSeeds []string `yaml:"excluded_seeds"`

	// SoilClusters provides a mapping between Gardener seed clusters and
	// soils.
	SoilClusters GardenerSoilClustersConfig `yaml:"soil_clusters"`
}

// GardenerSoilClustersConfig provides a mapping between Gardener seed clusters
// and soils.
type GardenerSoilClustersConfig struct {
	// GCP specifies the name of the GCP regional soil cluster.
	GCP string `yaml:"gcp"`
}

// DashboardConfig provides the Dashboard service configuration.
type DashboardConfig struct {
	// Address specifies the address on which the services binds
	Address string `yaml:"address"`

	// ReadOnly specifies whether to run the Dashboard UI in read-only mode.
	ReadOnly bool `yaml:"read_only"`

	// PrometheusEndpoint specifies the Prometheus endpoint from which the
	// Dashboard UI will read metrics.
	PrometheusEndpoint string `yaml:"prometheus_endpoint"`
}

// LoggingConfig provides the logging-specific settings.
type LoggingConfig struct {
	// Format specifies the output format.
	Format string `yaml:"format"`

	// AddSource specifies whether to include source code position for the
	// logging statements.
	AddSource bool `yaml:"add_source"`

	// Level specifies the logging level.
	Level string `yaml:"level"`

	// Attributes provides a default set of key/value pairs to be added to
	// each log event.
	Attributes map[string]string `yaml:"attributes"`
}

// ParseFileInto parses the configuration from the given path and unmarshals it
// into the specified out value.
func ParseFileInto(path string, out any) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	return nil
}

// Parse parses the configs from the given paths in-order. Configuration
// settings provided later in the sequence of paths will override settings from
// previous config paths.
func Parse(paths ...string) (*Config, error) {
	var conf Config

	for _, path := range paths {
		// Ignore empty paths
		if path == "" {
			continue
		}

		if err := ParseFileInto(path, &conf); err != nil {
			return nil, err
		}

		if conf.Version == "" {
			return nil, fmt.Errorf("%w: %s", ErrNoConfigVersion, path)
		}

		if conf.Version != ConfigFormatVersion {
			return nil, fmt.Errorf("%w: %s (%s)", ErrUnsupportedVersion, conf.Version, path)
		}
	}

	// AWS defaults
	if conf.AWS.AppID == "" {
		conf.AWS.AppID = DefaultAWSAppID
	}

	return &conf, nil
}

// MustParse parses the configs from the given paths, or panics in case of
// errors.
func MustParse(paths ...string) *Config {
	config, err := Parse(paths...)
	if err != nil {
		panic(err)
	}

	return config
}
