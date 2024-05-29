package config

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
