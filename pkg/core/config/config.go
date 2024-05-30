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

	// RetentionConfig represents the retention configuration.
	Retention RetentionConfig `yaml:"retention"`
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

// RetentionConfig provides retention specific configuration settings.
type RetentionConfig struct {
	// Interval specifies the periodic interval at which to run housekeeping
	// activities.
	Interval string `yaml:"interval"`

	// Models specifies the list of models to cleanup.
	Models []*ModelRetentionConfig `yaml:"models"`
}

// ModelRetentionConfig represents the retention configuration for a given
// model.
type ModelRetentionConfig struct {
	// Name specifies the model name.
	Name string `yaml:"name"`

	// Duration specifies the max duration for which an object will be kept,
	// if it hasn't been updated recently.
	//
	// For example:
	//
	// UpdatedAt field for an object is set to: Thu May 30 16:00:00 EEST 2024
	// Duration of the object is configured to: 4 hours
	//
	// If the object is not update anymore by the time the housekeeper runs,
	// after 20:00:00 this object will be considered as stale and removed
	// from the database.
	Duration time.Duration `yaml:"duration"`
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
