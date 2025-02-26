// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	dbclient "github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/core/registry"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
)

// NewWorkerCommand returns a new command for interfacing with the workers.
func NewWorkerCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "worker",
		Usage:   "worker operations",
		Aliases: []string{"w"},
		Before: func(ctx *cli.Context) error {
			conf := getConfig(ctx)
			return validateRedisConfig(conf)
		},
		Subcommands: []*cli.Command{
			{
				Name:    "list",
				Usage:   "list running workers",
				Aliases: []string{"ls"},
				Action: func(ctx *cli.Context) error {
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close()
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
						"PROCESSING",
						"UPTIME",
						"QUEUES",
					}
					table := newTableWriter(os.Stdout, headers)

					for _, item := range servers {
						uptime := time.Since(item.Started)
						queuesInfo := make([]string, 0)
						queueNames := slices.Sorted(maps.Keys(item.Queues))
						for _, qname := range queueNames {
							queuesInfo = append(queuesInfo, fmt.Sprintf("%s:%d", qname, item.Queues[qname]))
						}
						row := []string{
							item.Host,
							strconv.Itoa(item.PID),
							strconv.Itoa(item.Concurrency),
							item.Status,
							strconv.Itoa(len(item.ActiveWorkers)),
							uptime.String(),
							strings.Join(queuesInfo, ","),
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
					conf := getConfig(ctx)
					inspector := newInspector(conf)
					defer inspector.Close()
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
				Name:    "start",
				Usage:   "start worker",
				Aliases: []string{"s"},
				Before: func(ctx *cli.Context) error {
					conf := getConfig(ctx)
					validatorFuncs := []func(c *config.Config) error{
						validateWorkerConfig,
						validateDBConfig,
					}

					for _, validator := range validatorFuncs {
						if err := validator(conf); err != nil {
							return err
						}
					}

					return nil
				},
				Action: func(ctx *cli.Context) error {
					conf := getConfig(ctx)
					db := newDB(conf)
					defer db.Close()
					client := newAsynqClient(conf)
					defer client.Close()
					server := newServer(conf)
					inspector := newInspector(conf)
					defer inspector.Close()

					// Gardener client configs
					if err := configureGardenerClient(ctx.Context, conf); err != nil {
						return err
					}

					// Initialize DB and asynq client
					slog.Info("configuring db client")
					dbclient.SetDB(db)

					slog.Info("configuring asynq client")
					asynqclient.SetClient(client)

					// Initialize async inspector
					slog.Info("configuring asynq inspector")
					asynqclient.SetInspector(inspector)

					if err := configureAWSClients(ctx.Context, conf); err != nil {
						return err
					}

					if err := configureGCPClients(ctx.Context, conf); err != nil {
						return err
					}
					defer closeGCPClients()

					if err := configureAzureClients(ctx.Context, conf); err != nil {
						return err
					}

					if err := configureOpenStackClients(ctx.Context, conf); err != nil {
						return err
					}

					// Configure logging and middlewares
					slog.Info("configuring logging and middlewares")
					logger, err := newLogger(os.Stdout, conf)
					if err != nil {
						return err
					}
					middlewares := []asynq.MiddlewareFunc{
						asynqutils.NewLoggerMiddleware(logger),
						asynqutils.NewMeasuringMiddleware(),
					}

					// Register our task handlers
					mux := asynq.NewServeMux()
					mux.Use(middlewares...)

					walker := func(name string, handler asynq.Handler) error {
						slog.Info("registering task", "name", name)
						mux.Handle(name, handler)
						return nil
					}

					if err := registry.TaskRegistry.Range(walker); err != nil {
						return err
					}

					slog.Info("worker concurrency", "level", conf.Worker.Concurrency)
					slog.Info("queue priority", "strict", conf.Worker.StrictPriority)
					for queue, priority := range conf.Worker.Queues {
						slog.Info("queue configuration", "name", queue, "priority", priority)
					}

					return server.Run(mux)
				},
			},
		},
	}

	return cmd
}
