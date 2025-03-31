// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"

	"github.com/hibiken/asynq"
	"github.com/olekukonko/tablewriter"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/internal/pkg/migrations"
	"github.com/gardener/inventory/pkg/core/config"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	workerutils "github.com/gardener/inventory/pkg/utils/asynq/worker"
	dbutils "github.com/gardener/inventory/pkg/utils/db"
	slogutils "github.com/gardener/inventory/pkg/utils/slog"
)

// na is the const used to represent N/A values
const na = "N/A"

// configKey is the key used to store the parsed configuration in the context
type configKey struct{}

// errNoDashboardAddress is an error, which is returned when the Dashboard
// service was not configured with a bind address.
var errNoDashboardAddress = errors.New("no bind address specified")

// errNoServiceCredentials is an error, which is returned when a cloud provider
// API service (e.g. AWS, GCP, etc.)  does not have any named credentials
// configured.
var errNoServiceCredentials = errors.New("no credentials specified for service")

// errUnknownNamedCredentials is an error which is returned when a service is
// using an unknown named credentials.
var errUnknownNamedCredentials = errors.New("unknown named credentials")

// errNoAuthenticationMethod is an error, which is returned when no
// authentication method was specified in named credentials.
var errNoAuthenticationMethod = errors.New("no authentication method specified")

// errUnknownAuthenticationMethod is an error, which is returned when using an
// unknown/unsupported authentication method in named credentials.
var errUnknownAuthenticationMethod = errors.New("unknown authentication method specified")

// getConfig extracts and returns the [config.Config] from app's context.
func getConfig(ctx *cli.Context) *config.Config {
	conf := ctx.Context.Value(configKey{}).(*config.Config)
	return conf
}

// validateDashboardConfig validates the Dashboard service configuration.
func validateDashboardConfig(conf *config.Config) error {
	if conf.Dashboard.Address == "" {
		return errNoDashboardAddress
	}

	_, err := url.Parse(conf.Dashboard.PrometheusEndpoint)
	return err
}

// newLogger creates a new [slog.Logger] based on the provided [config.Config]
// spec, which outputs to the given [io.Writer].
func newLogger(w io.Writer, conf *config.Config) (*slog.Logger, error) {
	return slogutils.NewFromConfig(w, conf.Logging)
}

// newRedisClientOpt returns a new [asynq.RedisClientOpt] from the given config.
func newRedisClientOpt(conf *config.Config) asynq.RedisClientOpt {
	return asynqutils.NewRedisClientOptFromConfig(conf.Redis)
}

// newAsynqClient creates a new [asynq.Client] from the given config
func newAsynqClient(conf *config.Config) *asynq.Client {
	redisClientOpt := newRedisClientOpt(conf)
	return asynq.NewClient(redisClientOpt)
}

// newInspector returns a new [asynq.Inspector] from the given config.
func newInspector(conf *config.Config) *asynq.Inspector {
	redisClientOpt := newRedisClientOpt(conf)
	return asynq.NewInspector(redisClientOpt)
}

// newWorker creates a new [workerutils.Worker] from the given config.
func newWorker(conf *config.Config) *workerutils.Worker {
	redisClientOpt := newRedisClientOpt(conf)
	opts := make([]workerutils.Option, 0)
	logLevel := asynq.InfoLevel
	if conf.Debug {
		logLevel = asynq.DebugLevel
	}

	opts = append(opts, workerutils.WithLogLevel(logLevel))
	opts = append(opts, workerutils.WithErrorHandler(asynqutils.NewDefaultErrorHandler()))
	worker := workerutils.NewFromConfig(redisClientOpt, conf.Worker, opts...)

	// Configure middlewares
	middlewares := []asynq.MiddlewareFunc{
		asynqutils.NewLoggerMiddleware(slog.Default()),
		asynqutils.NewMeasuringMiddleware(),
	}
	worker.UseMiddlewares(middlewares...)

	return worker
}

// newDB returns a new [bun.DB] database from the given config.
func newDB(conf *config.Config) (*bun.DB, error) {
	db, err := dbutils.NewFromConfig(conf.Database)
	if err != nil {
		return nil, err
	}
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(conf.Debug)))

	return db, nil
}

// newMigrator creates a new [github.com/uptrace/bun/migrate.Migrator] from the
// given config.
func newMigrator(conf *config.Config, db *bun.DB) (*migrate.Migrator, error) {
	// By default we will use the bundled migrations, unless we have an
	// explicitly specified alternate migrations directory.
	m := migrations.Migrations
	migrationDir := conf.Database.MigrationDirectory

	if migrationDir != "" {
		m = migrate.NewMigrations(migrate.WithMigrationsDirectory(migrationDir))
		err := m.Discover(os.DirFS(migrationDir))
		switch {
		case err == nil:
			break
		case errors.Is(err, fs.ErrNotExist):
			slog.Warn(
				"falling back to bundled migrations",
				"reason", "migration path does not exist",
				"path", migrationDir,
			)
			m = migrations.Migrations
		default:
			// Any other error should bubble up to the caller
			return nil, fmt.Errorf("failed to discover migrations from %s: %w", migrationDir, err)
		}
	}

	migrator := migrate.NewMigrator(db, m, migrate.WithMarkAppliedOnSuccess(true))
	return migrator, nil
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

	if conf.Scheduler.DefaultQueue == "" {
		conf.Scheduler.DefaultQueue = config.DefaultQueueName
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
