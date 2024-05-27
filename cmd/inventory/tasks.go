package main

import (
	// Imports only for their side effects of registering our tasks
	"log/slog"
	"os"

	"github.com/gardener/inventory/pkg/aws/tasks"
	_ "github.com/gardener/inventory/pkg/aws/tasks"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"
)

func NewTaskCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "task",
		Usage:   "task type",
		Aliases: []string{"t"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "redis-endpoint",
				Usage:    "Redis endpoint to connect to",
				EnvVars:  []string{"REDIS_ENDPOINT"},
				Required: true,
			},
		},
		Subcommands: []*cli.Command{
			{
				Name:  "enqueue",
				Usage: "enqueue asynq task",
				Action: func(ctx *cli.Context) error {
					asynqClient := newAsynqClientFromFlags(ctx)
					defer asynqClient.Close()
					var task *asynq.Task
					switch ctx.Args().First() {
					case tasks.AWS_COLLECT_REGIONS_TYPE:
						task = tasks.NewAwsCollectRegionsTask()
					case tasks.AWS_COLLECT_AZS_TYPE:
						task = tasks.NewCollectAzsTask()
					default:
						slog.Error("unknown task type", "type", ctx.Args().First())
					}
					if task == nil {
						os.Exit(1)
					}

					info, err := asynqClient.Enqueue(task)
					if err != nil {
						slog.Error("could not enqueu task", "type", task.Type(), "err", err)
						os.Exit(1)
					}
					slog.Info("enqueued task", "type", task.Type(), "id", info.ID, "queue", info.Queue)

					return nil
				},
			},
		},
	}

	return cmd
}

func newAsynqClientFromFlags(ctx *cli.Context) *asynq.Client {
	redisEndpoint := ctx.String("redis-endpoint")

	// TODO: Handle authentication, TLS, etc.
	redisClientOpt := asynq.RedisClientOpt{
		Addr: redisEndpoint,
	}
	return asynq.NewClient(redisClientOpt)

}
