// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
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

	// Redis represents the Redis configuration
	Redis RedisConfig `yaml:"redis"`

	// Database represents the database configuration.
	Database DatabaseConfig `yaml:"database"`

	// Worker represents the worker configuration.
	Worker WorkerConfig `yaml:"worker"`

	// Scheduler represents the scheduler configuration.
	Scheduler SchedulerConfig `yaml:"scheduler"`

	// VirtualGarden represents the virtual garden configuration.
	VirtualGarden VirtualGardenConfig `yaml:"virtual_garden"`

	// Dashboard represents the configuration for the Dashboard
	// service.
	Dashboard DashboardConfig `yaml:"dashboard"`

	// AWS represents the AWS specific configuration settings.
	AWS AWSConfig `yaml:"aws"`
}

// AWSConfig provides AWS specific configuration settings.
type AWSConfig struct {
	// Region is the region to use when initializing the AWS client.
	Region string `yaml:"region"`

	// DefaultRegion is the default region to use when initializing the AWS client.
	DefaultRegion string `yaml:"default_region"`

	// AppID is an optional application specific identifier.
	AppID string `yaml:"app_id"`

	// Credentials specifies the AWS credentials configuration.
	Credentials AWSCredentialsConfig `yaml:"credentials"`
}

// AWSCredentialsConfig provides credentials specific configuration for the AWS
// client.
type AWSCredentialsConfig struct {
	// Provider represents the credentials provider. Currently supported
	// providers are `default' and `kube_sa_token'.
	//
	// When using the `default' credentials provider the AWS client will be
	// initialized using the shared credentials file at ~/.aws/credentials.
	//
	// With the `kube_sa_token' credentials provider the AWS client will be
	// initialized with Web Identity Credentials Provider, which uses
	// Kubernetes service account tokens, which are then exchanged for
	// temporary security credentials when communicating with the AWS
	// services.
	Provider string `yaml:"provider"`

	// KubeSATokenProvider provides the configuration settings for the
	// Kubernetes Service Account Token Credentials Provider.
	KubeSATokenProvider AWSKubeSATokenProviderConfig `yaml:"kube_sa_token"`
}

// AWSKubeSATokenProviderConfig represents the configuration settings for the
// AWS Kubernetes Service Account Token credentials provider.
type AWSKubeSATokenProviderConfig struct {
	// Kubeconfig specifies the path to a Kubeconfig file to use when
	// creating the underlying Kubernetes client. If empty, the Kubernetes
	// client will be created using in-cluster configuration.
	Kubeconfig string `yaml:"kubeconfig"`

	// ServiceAccount specifies the Kubernetes service account name.
	ServiceAccount string `yaml:"service_account"`

	// Namespace specifies the Kubernetes namespace of the service account.
	Namespace string `yaml:"namespace"`

	// Duration specifies the expiry duration for the service account token.
	Duration time.Duration `yaml:"duration"`

	// Audiences specifies the list of audiences the service account token
	// will be issued for.
	Audiences []string `yaml:"audiences"`

	// RoleARN specifies the IAM Role ARN to be assumed.
	RoleARN string `yaml:"role_arn"`

	// RoleSessionName is a unique name for the session.
	RoleSessionName string `yaml:"role_session_name"`
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
	// Concurrency specifies the concurrency level for workers.
	Concurrency int `yaml:"concurrency"`
}

// SchedulerConfig provides scheduler specific configuration settings.
type SchedulerConfig struct {
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
}

// VirtualGardenConfig represents the Virtual Garden configuration.
type VirtualGardenConfig struct {
	// KubeConfig is the path to the kubeconfig file of the Virtual Garden
	Kubeconfig string `yaml:"kubeconfig"`
	// Environment is the environment of the running Garden
	Environment string `yaml:"environment"`
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

// Parse parses the config from the given path.
func Parse(path string) (*Config, error) {
	var conf Config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	if conf.Version == "" {
		return nil, ErrNoConfigVersion
	}

	if conf.Version != ConfigFormatVersion {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedVersion, conf.Version)
	}

	// Set defaults
	if conf.Worker.Concurrency <= 0 {
		conf.Worker.Concurrency = runtime.NumCPU()
	}

	return &conf, nil
}

// MustParse parses the config from the given path, or panics in case of errors.
func MustParse(path string) *Config {
	config, err := Parse(path)
	if err != nil {
		panic(err)
	}

	return config
}
