package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the Inventory configuration.
type Config struct {
	// Version is the version of the config file.
	Version string `yaml:"version"`

	// Redis represents the Redis configuration
	Redis RedisConfig `yaml:"redis"`

	// Database represents the database configuration.
	Database DatabaseConfig `yaml:"database"`

	// Worker represents the worker configuration.
	Worker WorkerConfig `yaml:"worker"`
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

// Parse parses the config from the given path.
func Parse(path string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// MustParse parses the config from the given path, or panics in case of errors.
func MustParse(path string) *Config {
	config, err := Parse(path)
	if err != nil {
		panic(err)
	}

	return config
}
