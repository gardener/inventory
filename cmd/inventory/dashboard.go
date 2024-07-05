// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log/slog"
	"net/http"

	"github.com/gardener/inventory/pkg/core/config"
	"github.com/hibiken/asynq/x/metrics"
	"github.com/hibiken/asynqmon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
)

// NewDashboardCommand returns a new command for interfacing with the dashboard.
func NewDashboardCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "dashboard",
		Usage:   "dashboard operations",
		Aliases: []string{"ui"},
		Before: func(ctx *cli.Context) error {
			conf := getConfig(ctx)
			validatorFuncs := []func(c *config.Config) error{
				validateRedisConfig,
				validateDashboardConfig,
			}

			for _, validator := range validatorFuncs {
				if err := validator(conf); err != nil {
					return err
				}
			}

			return nil
		},
		Subcommands: []*cli.Command{
			{
				Name:    "start",
				Usage:   "start the dashboard ui",
				Aliases: []string{"s"},
				Action: func(ctx *cli.Context) error {
					conf := getConfig(ctx)
					redisClientOpt := newRedisClientOpt(conf)
					inspector := newInspector(conf)

					// Asynq UI
					opts := asynqmon.Options{
						RootPath:          "/",
						RedisConnOpt:      redisClientOpt,
						ReadOnly:          conf.Dashboard.ReadOnly,
						PrometheusAddress: conf.Dashboard.PrometheusEndpoint,
					}
					ui := asynqmon.New(opts)

					// Metrics
					promRegistry := prometheus.NewPedanticRegistry()
					promRegistry.MustRegister(
						// Queue metrics
						metrics.NewQueueMetricsCollector(inspector),
						// Standard Go metrics
						collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
						collectors.NewGoCollector(),
					)

					mux := http.NewServeMux()
					mux.Handle("/", ui)
					mux.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))

					srv := &http.Server{
						Addr:    conf.Dashboard.Address,
						Handler: mux,
					}

					slog.Info("starting server", "address", conf.Dashboard.Address, "ui", "/", "metrics", "/metrics")

					return srv.ListenAndServe()
				},
			},
		},
	}

	return cmd
}
