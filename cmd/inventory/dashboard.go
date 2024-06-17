// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log/slog"
	"net/http"

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
			return validateRedisConfig(conf)
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "bind server to this address",
				Value:   ":8080",
				EnvVars: []string{"ADDRESS"},
			},
			&cli.BoolFlag{
				Name:    "read-only",
				Usage:   "if set to true, ui will run in read-only mode",
				Value:   false,
				EnvVars: []string{"READ_ONLY"},
			},
			&cli.StringFlag{
				Name:    "prometheus-endpoint",
				Usage:   "prometheus endpoint to query data from",
				Value:   "",
				EnvVars: []string{"PROMETHEUS_ENDPOINT"},
			},
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

					address := ctx.String("address")
					readOnly := ctx.Bool("read-only")
					prometheusEndpoint := ctx.String("prometheus-endpoint")

					// Asynq UI
					opts := asynqmon.Options{
						RootPath:          "/",
						RedisConnOpt:      redisClientOpt,
						ReadOnly:          readOnly,
						PrometheusAddress: prometheusEndpoint,
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
						Addr:    address,
						Handler: mux,
					}

					slog.Info("starting server", "address", address, "ui", "/", "metrics", "/metrics")

					return srv.ListenAndServe()
				},
			},
		},
	}

	return cmd
}
