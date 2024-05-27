package main

import (
	"log/slog"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"
)

// NewSchedulerCommand returns a new command for interfacing with the scheduler.
func NewSchedulerCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "scheduler",
		Usage:   "scheduler operations",
		Aliases: []string{"s"},
		Subcommands: []*cli.Command{
			{
				Name:  "start",
				Usage: "start the scheduler",
				Action: func(ctx *cli.Context) error {
					scheduler := newSchedulerFromFlags(ctx)

					// Register our periodic tasks
					registry.ScheduledTaskRegistry.Range(func(spec string, task *asynq.Task) error {
						slog.Info("registering periodic task", "spec", spec, "name", task.Type())
						scheduler.Register(spec, task)
						return nil
					})

					if err := scheduler.Run(); err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	return cmd
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
