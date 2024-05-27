package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/gardener/inventory/internal/pkg/migrations"
	"github.com/hibiken/asynq"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
)

// newInspectorFromFlags returns a new [asynq.Inspector] from the specified
// flags.
func newInspectorFromFlags(ctx *cli.Context) *asynq.Inspector {
	redisEndpoint := ctx.String("redis-endpoint")
	redisClientOpt := asynq.RedisClientOpt{
		Addr: redisEndpoint,
	}

	return asynq.NewInspector(redisClientOpt)
}

// newAsynqServerFromFlags creates a new [asynq.Server] from the specified
// flags.
func newAsynqServerFromFlags(ctx *cli.Context) *asynq.Server {
	redisEndpoint := ctx.String("redis-endpoint")
	concurrency := ctx.Int("concurrency")

	// TODO: Handle authentication, TLS, etc.
	redisClientOpt := asynq.RedisClientOpt{
		Addr: redisEndpoint,
	}

	// TODO: Logger, priority queues, log level, etc.
	config := asynq.Config{
		Concurrency: concurrency,
		BaseContext: func() context.Context { return ctx.Context },
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
		m.Discover(os.DirFS(migrationDir))
	}

	return migrate.NewMigrator(db, m)
}

// newSchedulerFromFlags creates a new [asynq.Scheduler] from the specified
// flags.
func newSchedulerFromFlags(ctx *cli.Context) *asynq.Scheduler {
	redisEndpoint := ctx.String("redis-endpoint")

	// TODO: Handle authentication, TLS, etc.
	redisClientOpt := asynq.RedisClientOpt{
		Addr: redisEndpoint,
	}

	// TODO: Logger, log level, etc.
	preEnqueueFunc := func(t *asynq.Task, opts []asynq.Option) {
		slog.Info("enqueueing task", "name", t.Type())
	}

	opts := &asynq.SchedulerOpts{
		PreEnqueueFunc: preEnqueueFunc,
	}

	scheduler := asynq.NewScheduler(redisClientOpt, opts)
	return scheduler
}
