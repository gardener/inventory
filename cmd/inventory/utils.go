package main

import (
	"context"
	"database/sql"
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

// getConfig extracts and returns the [config.Config] from app's context.
func getConfig(ctx *cli.Context) *config.Config {
	conf := ctx.Context.Value(configKey{}).(*config.Config)
	return conf
}

// newRedisClientOpt returns a new [asynq.RedisClientOpt] from the specified
// flags.
func newRedisClientOpt(ctx *cli.Context) asynq.RedisClientOpt {
	// TODO: Handle authentication, TLS, etc.
	endpoint := ctx.String("redis-endpoint")
	opts := asynq.RedisClientOpt{
		Addr: endpoint,
	}

	return opts
}

// newInspectorFromFlags returns a new [asynq.Inspector] from the specified
// flags.
func newInspectorFromFlags(ctx *cli.Context) *asynq.Inspector {
	redisClientOpt := newRedisClientOpt(ctx)
	return asynq.NewInspector(redisClientOpt)
}

// newAsynqServerFromFlags creates a new [asynq.Server] from the specified
// flags.
func newAsynqServerFromFlags(ctx *cli.Context) *asynq.Server {
	debug := ctx.Bool("debug")
	concurrency := ctx.Int("concurrency")
	redisClientOpt := newRedisClientOpt(ctx)

	// TODO: Logger, priority queues, etc.
	logLevel := asynq.InfoLevel
	if debug {
		logLevel = asynq.DebugLevel
	}

	config := asynq.Config{
		Concurrency: concurrency,
		BaseContext: func() context.Context { return ctx.Context },
		LogLevel:    logLevel,
	}

	server := asynq.NewServer(redisClientOpt, config)

	return server
}

// newDbFromFlags returns a Bun database from the specified flags
func newDBFromFlags(ctx *cli.Context) *bun.DB {
	dsn := ctx.String("dsn")
	debug := ctx.Bool("debug")

	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(pgdb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(debug)))

	return db
}

// newMigratorFromFlags returns a new [github.com/uptrace/bun/migrate.Migrator]
// from the specified flags.
func newMigratorFromFlags(ctx *cli.Context, db *bun.DB) *migrate.Migrator {
	// By default we will use the bundled migrations, unless we have an
	// explicitely specified alternate migrations directory.
	m := migrations.Migrations
	migrationDir := ctx.String("migration-dir")
	if migrationDir != "" {
		m = migrate.NewMigrations(migrate.WithMigrationsDirectory(migrationDir))
		err := m.Discover(os.DirFS(migrationDir))
		if err != nil {
			slog.Error("failed to discover migrations", "error", err)
		}
	}

	return migrate.NewMigrator(db, m)
}

// newSchedulerFromFlags creates a new [asynq.Scheduler] from the specified
// flags.
func newSchedulerFromFlags(ctx *cli.Context) *asynq.Scheduler {
	debug := ctx.Bool("debug")
	redisClientOpt := newRedisClientOpt(ctx)

	// TODO: Logger, log level, etc.
	// TODO: PostEnqueue hook to emit metrics per tasks
	preEnqueueFunc := func(t *asynq.Task, opts []asynq.Option) {
		slog.Info("enqueueing task", "name", t.Type())
	}

	errEnqueueFunc := func(t *asynq.Task, opts []asynq.Option, err error) {
		slog.Error("failed to enqueue", "name", t.Type(), "error", err)
	}

	logLevel := asynq.InfoLevel
	if debug {
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

// newAsynqClientFromFlags creates a new [asynq.Client]
func newAsynqClientFromFlags(ctx *cli.Context) *asynq.Client {
	redisClientOpt := newRedisClientOpt(ctx)
	return asynq.NewClient(redisClientOpt)
}
