// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
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
	"github.com/gardener/inventory/pkg/core/registry"
)

// NewWorkerCommand returns a new command for interfacing with the workers.
func NewWorkerCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "worker",
		Usage:   "worker operations",
		Aliases: []string{"w"},
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
						Required: false,
						Aliases:  []string{"name"},
					},
					&cli.BoolFlag{
						Name:  "local",
						Usage: "ping local workers",
					},
				},
				Action: func(ctx *cli.Context) error {
					// Note: currently asynq does not expose Ping() methods for connected
					// workers, but we can still rely on the [asynq.Inspector.Servers] to
					// view whether a given worker is up and running.
					workerName := ctx.String("worker")
					localWorker := ctx.Bool("local")

					switch {
					case workerName == "" && !localWorker:
						return errors.New("must specify either --worker or --local flag")
					case workerName != "" && localWorker:
						return errors.New("cannot specify --worker and --local at the same time")
					case localWorker:
						hostname, err := os.Hostname()
						if err != nil {
							return err
						}
						workerName = hostname
					}

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
				Action: func(ctx *cli.Context) error {
					conf := getConfig(ctx)
					db, err := newDB(conf)
					if err != nil {
						return err
					}
					defer db.Close()
					client := newAsynqClient(conf)
					defer client.Close()
					inspector := newInspector(conf)
					defer inspector.Close()
					worker := newWorker(conf)

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

					// Register our task handlers using the default registry
					worker.HandlersFromRegistry(registry.TaskRegistry)
					_ = registry.TaskRegistry.Range(func(name string, _ asynq.Handler) error {
						slog.Info("registered task", "name", name)
						return nil
					})

					slog.Info("worker concurrency", "level", conf.Worker.Concurrency)
					slog.Info("queue priority", "strict", conf.Worker.StrictPriority)
					for queue, priority := range conf.Worker.Queues {
						slog.Info("queue configuration", "name", queue, "priority", priority)
					}

					return worker.Run()
				},
			},
		},
	}

	return cmd
}
