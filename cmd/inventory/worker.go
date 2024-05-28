package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gardener/inventory/pkg/aws/clients"

	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"
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
		},
		Subcommands: []*cli.Command{
			{
				Name:    "list",
				Usage:   "list running workers",
				Aliases: []string{"ls"},
				Action: func(ctx *cli.Context) error {
					inspector := newInspectorFromFlags(ctx)
					servers, err := inspector.Servers()
					if err != nil {
						return err
					}

					if len(servers) == 0 {
						return nil
					}

					headers := []string{
						"HOST",
						"PID",
						"CONCURRENCY",
						"STATUS",
						"UPTIME",
					}
					table := newTableWriter(os.Stdout, headers)

					for _, item := range servers {
						uptime := time.Since(item.Started)
						row := []string{
							item.Host,
							strconv.Itoa(item.PID),
							strconv.Itoa(item.Concurrency),
							item.Status,
							uptime.String(),
						}
						table.Append(row)
					}

					table.Render()

					return nil
				},
			},
			{
				Name:    "ping",
				Usage:   "ping a worker",
				Aliases: []string{"p"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "worker",
						Usage:    "worker name to ping",
						Required: true,
						Aliases:  []string{"name"},
					},
				},
				Action: func(ctx *cli.Context) error {
					// Note: currently asynq does not expose Ping() methods for connected
					// workers, but we can still rely on the [asynq.Inspector.Servers] to
					// view whether a given worker is up and running.
					workerName := ctx.String("worker")
					inspector := newInspectorFromFlags(ctx)
					servers, err := inspector.Servers()
					if err != nil {
						return err
					}

					exists := false
					for _, item := range servers {
						if item.Host == workerName {
							exists = true
							fmt.Printf("%s/%d: OK\n", item.Host, item.PID)
						}
					}

					if !exists {
						return cli.Exit("", 1)
					}

					return nil
				},
			},
			{
				Name:  "start",
				Usage: "start worker",
				Action: func(ctx *cli.Context) error {
					server := newAsynqServerFromFlags(ctx)
					mux := asynq.NewServeMux()

					// Initialize clients in workers
					clients.SetDB(newDBFromFlags(ctx))
					clients.SetClient(newAsynqClientFromFlags(ctx))

					// Register our task handlers
					registry.TaskRegistry.Range(func(name string, handler asynq.Handler) error {
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
