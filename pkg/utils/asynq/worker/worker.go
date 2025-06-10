// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/metrics"
)

// Option is a function, which configures the [Worker].
type Option func(conf *asynq.Config)

// Worker wraps an [asynq.Server] and [asynq.ServeMux] with additional
// convenience methods for task handlers. It also provides an HTTP server, which
// serves worker-related metrics.
type Worker struct {
	asynqServer   *asynq.Server
	asynqMux      *asynq.ServeMux
	metricsAddr   string
	metricsPath   string
	metricsServer *http.Server
}

// WithLogLevel is an [Option], which configures the log level of the [Worker].
func WithLogLevel(level asynq.LogLevel) Option {
	opt := func(conf *asynq.Config) {
		conf.LogLevel = level
	}

	return opt
}

// WithErrorHandler is an [Option], which configures the [Worker] to use the
// specified [asynq.ErrorHandler].
func WithErrorHandler(handler asynq.ErrorHandler) Option {
	opt := func(conf *asynq.Config) {
		conf.ErrorHandler = handler
	}

	return opt
}

// NewFromConfig creates a new [Worker] based on the provided
// [config.WorkerConfig] spec.
func NewFromConfig(r asynq.RedisClientOpt, conf config.WorkerConfig, opts ...Option) *Worker {
	concurrency := conf.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	defaultQueues := map[string]int{
		config.DefaultQueueName: 1,
	}

	queues := conf.Queues
	if len(queues) == 0 {
		queues = defaultQueues
	}

	asynqConfig := asynq.Config{
		Concurrency:    concurrency,
		Queues:         queues,
		StrictPriority: conf.StrictPriority,
	}

	for _, opt := range opts {
		opt(&asynqConfig)
	}

	metricsAddr := conf.Metrics.Address
	if metricsAddr == "" {
		metricsAddr = config.DefaultWorkerMetricsAddress
	}

	metricsPath := conf.Metrics.Path
	if metricsPath == "" {
		metricsPath = config.DefaultWorkerMetricsPath
	}

	asynqServer := asynq.NewServer(r, asynqConfig)
	asynqMux := asynq.NewServeMux()
	metricsServer := metrics.NewServer(metricsAddr, metricsPath)

	worker := &Worker{
		asynqServer:   asynqServer,
		asynqMux:      asynqMux,
		metricsAddr:   metricsAddr,
		metricsPath:   metricsPath,
		metricsServer: metricsServer,
	}

	return worker
}

// UseMiddlewares configures the [Worker] multiplexer to use the specified
// [asynq.MiddlewareFunc].
func (w *Worker) UseMiddlewares(middlewares ...asynq.MiddlewareFunc) {
	w.asynqMux.Use(middlewares...)
}

// Handle registers a new task handler with the [Worker]'s multiplexer.
func (w *Worker) Handle(pattern string, handler asynq.Handler) {
	w.asynqMux.Handle(pattern, handler)
}

// HandlersFromRegistry registers task handlers with the [Worker] multiplexer
// using the given registry.
func (w *Worker) HandlersFromRegistry(reg *registry.Registry[string, asynq.Handler]) {
	_ = reg.Range(func(pattern string, handler asynq.Handler) error {
		w.Handle(pattern, handler)

		return nil
	})
}

// Run starts the task processing by calling [asynq.Server.Start] and blocks
// until an OS signal is received.
func (w *Worker) Run() error {
	go func() {
		slog.Info(
			"starting metrics server",
			"address", w.metricsAddr,
			"path", w.metricsPath,
		)
		if err := w.metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("failed to start metrics server", "reason", err)
		}
	}()

	return w.asynqServer.Run(w.asynqMux)
}

// Shutdown gracefully shuts down the server by calling [asynq.Server.Shutdown].
func (w *Worker) Shutdown() {
	w.asynqServer.Shutdown()

	slog.Info("shutting down metrics server")
	if err := w.metricsServer.Shutdown(context.Background()); err != nil {
		slog.Error("failed to gracefully shutdown metrics server", "reason", err)
	}
}
