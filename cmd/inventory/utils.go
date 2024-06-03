package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"
	"github.com/olekukonko/tablewriter"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/internal/pkg/migrations"
	"github.com/gardener/inventory/pkg/core/config"
)

// configKey is the key used to store the parsed configuration in the context
type configKey struct{}

// errInvalidDSN error is returned, if the DSN configuration is incorrect, or
// empty.
var errInvalidDSN = errors.New("invalid or missing database configuration")

// errInvalidWorkerConcurrency error is returned when the worker concurrency
// setting is invalid, e.g. it is <= 0.
var errInvalidWorkerConcurrency = errors.New("invalid worker concurrency")

// errInvalidRedisEndpoint is returned when Redis is configured with an invalid
// endpoint.
var errInvalidRedisEndpoint = errors.New("invalid or missing redis endpoint")

// getConfig extracts and returns the [config.Config] from app's context.
func getConfig(ctx *cli.Context) *config.Config {
	conf := ctx.Context.Value(configKey{}).(*config.Config)
	return conf
}

// validateDBConfig validates the database configuration settings.
func validateDBConfig(conf *config.Config) error {
	if conf.Database.DSN == "" {
		return errInvalidDSN
	}

	return nil
}

// validateWorkerConfig validates the worker configuration settings.
func validateWorkerConfig(conf *config.Config) error {
	if conf.Worker.Concurrency <= 0 {
		return fmt.Errorf("%w: %d", errInvalidWorkerConcurrency, conf.Worker.Concurrency)
	}

	return nil
}

// validateRedisConfig validates the Redis configuration settings.
func validateRedisConfig(conf *config.Config) error {
	if conf.Redis.Endpoint == "" {
		return errInvalidRedisEndpoint
	}

	return nil
}

// newRedisClientOpt returns a new [asynq.RedisClientOpt] from the given config.
func newRedisClientOpt(conf *config.Config) asynq.RedisClientOpt {
	// TODO: Handle authentication, TLS, etc.
	opts := asynq.RedisClientOpt{
		Addr: conf.Redis.Endpoint,
	}

	return opts
}

// newClient creates a new [asynq.Client] from the given config
func newClient(conf *config.Config) *asynq.Client {
	redisClientOpt := newRedisClientOpt(conf)
	return asynq.NewClient(redisClientOpt)
}

// newInspector returns a new [asynq.Inspector] from the given config.
func newInspector(conf *config.Config) *asynq.Inspector {
	redisClientOpt := newRedisClientOpt(conf)
	return asynq.NewInspector(redisClientOpt)
}

// newServer creates a new [asynq.Server] from the given config.
func newServer(conf *config.Config) *asynq.Server {
	redisClientOpt := newRedisClientOpt(conf)

	// TODO: Logger, priority queues, etc.
	logLevel := asynq.InfoLevel
	if conf.Debug {
		logLevel = asynq.DebugLevel
	}

	errHandler := func(ctx context.Context, task *asynq.Task, err error) {
		taskID, _ := asynq.GetTaskID(ctx)
		queueName, _ := asynq.GetQueueName(ctx)
		retried, _ := asynq.GetRetryCount(ctx)
		slog.Error(
			"task failed",
			"id", taskID,
			"queue", queueName,
			"retry", retried,
			"reason", err,
		)
	}
	config := asynq.Config{
		Concurrency:  conf.Worker.Concurrency,
		LogLevel:     logLevel,
		ErrorHandler: asynq.ErrorHandlerFunc(errHandler),
	}

	server := asynq.NewServer(redisClientOpt, config)

	return server
}

// newDB returns a new [bun.DB] database from the given config.
func newDB(conf *config.Config) *bun.DB {
	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(conf.Database.DSN)))
	db := bun.NewDB(pgdb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(conf.Debug)))

	return db
}

// newMigrator creates a new [github.com/uptrace/bun/migrate.Migrator] from the
// given config.
func newMigrator(conf *config.Config, db *bun.DB) (*migrate.Migrator, error) {
	// By default we will use the bundled migrations, unless we have an
	// explicitely specified alternate migrations directory.
	m := migrations.Migrations
	migrationDir := conf.Database.MigrationDirectory
	if migrationDir != "" {
		m = migrate.NewMigrations(migrate.WithMigrationsDirectory(migrationDir))
		err := m.Discover(os.DirFS(migrationDir))
		if err != nil {
			return nil, fmt.Errorf("failed to discover migrations from %s: %w", migrationDir, err)
		}
	}

	return migrate.NewMigrator(db, m), nil
}

// newScheduler creates a new [asynq.Scheduler] from the given config.
func newScheduler(conf *config.Config) *asynq.Scheduler {
	redisClientOpt := newRedisClientOpt(conf)

	// TODO: Logger, etc.
	// TODO: PostEnqueue hook to emit metrics per tasks
	preEnqueueFunc := func(t *asynq.Task, opts []asynq.Option) {
		slog.Info("enqueueing task", "name", t.Type())
	}

	errEnqueueFunc := func(t *asynq.Task, opts []asynq.Option, err error) {
		slog.Error("failed to enqueue", "name", t.Type(), "error", err)
	}

	logLevel := asynq.InfoLevel
	if conf.Debug {
		logLevel = asynq.DebugLevel
	}

	opts := &asynq.SchedulerOpts{
		PreEnqueueFunc:      preEnqueueFunc,
		EnqueueErrorHandler: errEnqueueFunc,
		LogLevel:            logLevel,
	}

	scheduler := asynq.NewScheduler(redisClientOpt, opts)
	return scheduler
}

// newTableWriter creates a new [tablewriter.Table] with the given [io.Writer]
// and headers
func newTableWriter(w io.Writer, headers []string) *tablewriter.Table {
	table := tablewriter.NewWriter(w)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetBorder(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	return table
}
