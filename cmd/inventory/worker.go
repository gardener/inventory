// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/hibiken/asynq"
	"github.com/urfave/cli/v2"

	"github.com/gardener/inventory/pkg/clients"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	dbclient "github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/core/registry"
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
					conf := getConfig(ctx)
					inspector := newInspector(conf)
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
						validateAWSConfig,
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
					client := newClient(conf)
					server := newServer(conf)

					gardenConfigs, err := newGardenConfigs(conf)
					if err != nil {
						return err
					}

					awsConfig, err := loadAWSDefaultConfig(ctx.Context, conf)
					if err != nil {
						return err
					}
					ec2Client := ec2.NewFromConfig(awsConfig)
					elbClient := elb.NewFromConfig(awsConfig)

					// Initialize clients in workers
					dbclient.SetDB(db)
					asynqclient.SetClient(client)
					clients.SetGardenConfigs(gardenConfigs)
					clients.SetEC2Client(ec2Client)
					clients.SetELBClient(elbClient)

					// Register our task handlers
					mux := asynq.NewServeMux()
					mux.Use(newLoggingMiddleware())

					walker := func(name string, handler asynq.Handler) error {
						slog.Info("registering task", "name", name)
						mux.Handle(name, handler)
						return nil
					}

					if err := registry.TaskRegistry.Range(walker); err != nil {
						return err
					}

					slog.Info("worker concurrency", "level", conf.Worker.Concurrency)
					return server.Run(mux)
				},
			},
		},
	}

	return cmd
}
