// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"runtime"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/core/config"
	"github.com/gardener/inventory/pkg/core/registry"
)

// Option is a function, which configures the [Worker].
type Option func(conf *asynq.Config)

// Worker wraps an [asynq.Server] and [asynq.ServeMux] with additional
// convenience methods for task handlers.
type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
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

	config := asynq.Config{
		Concurrency:    concurrency,
		Queues:         queues,
		StrictPriority: conf.StrictPriority,
	}

	for _, opt := range opts {
		opt(&config)
	}

	server := asynq.NewServer(r, config)
	mux := asynq.NewServeMux()
	worker := &Worker{
		server: server,
		mux:    mux,
	}

	return worker
}

// UseMiddlewares configures the [Worker] multiplexer to use the specified
// [asynq.MiddlewareFunc].
func (w *Worker) UseMiddlewares(middlewares ...asynq.MiddlewareFunc) {
	w.mux.Use(middlewares...)
}

// Handle registers a new handler with the [Worker]'s multiplexer.
func (w *Worker) Handle(pattern string, handler asynq.Handler) {
	w.mux.Handle(pattern, handler)
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
	return w.server.Run(w.mux)
}

// Shutdown gracefully shuts down the server by calling [asynq.Server.Shutdown].
func (w *Worker) Shutdown() {
	w.server.Shutdown()
}
