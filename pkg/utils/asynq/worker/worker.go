// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"runtime"

	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/core/config"
)

// Option is a function, which configures the [Worker].
type Option func(conf *asynq.Config)

// Worker wraps an [asynq.Server] and [asynq.ServeMux] with additional
// convenience methods for task handlers.
type Worker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
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
