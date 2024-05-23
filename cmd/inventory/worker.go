package main

import (
	"context"
	"log/slog"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"

	// Import only for the side effects of registering our tasks
	_ "github.com/gardener/inventory/pkg/aws/tasks"
)

// NewWorkerCommand returns a new command for interfacing with the workers.
func NewWorkerCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "worker",
		Usage:   "worker operations",
		Aliases: []string{"w"},
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "concurrency",
				Usage:   "number of concurrent workers to start",
				EnvVars: []string{"CONCURRENCY_LEVEL"},
				Value:   10,
			},
			&cli.StringFlag{
				Name:     "redis-endpoint",
				Usage:    "Redis endpoint to connect to",
				EnvVars:  []string{"REDIS_ENDPOINT"},
				Required: true,
			},
		},
		Subcommands: []*cli.Command{
			{
				Name:  "start",
				Usage: "start the workers",
				Action: func(ctx *cli.Context) error {
					server := newAsynqServerFromFlags(ctx)
					mux := asynq.NewServeMux()

					// Register our task handlers
					registry.Range(func(name string, handler asynq.Handler) error {
						slog.Info("registering task", "name", name)
						mux.Handle(name, handler)
						return nil
					})

					if err := server.Run(mux); err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	return cmd
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
