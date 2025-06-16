// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Namespace is the namespace component of the fully qualified metric name
const Namespace = "inventory"

// DefaultRegistry is the default [prometheus.Registry] for metrics.
var DefaultRegistry = prometheus.NewPedanticRegistry()

var (
	// TaskSuccessfulTotal is a metric, which gets incremented each time a
	// task has been successfully executed.
	TaskSuccessfulTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "task_successful_total",
			Help:      "Total number of times a task has been successfully executed",
		},
		[]string{"task_name", "task_queue"},
	)

	// TaskFailedTotal is a metric, which gets incremented each time a task
	// has failed.
	TaskFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "task_failed_total",
			Help:      "Total number of times a task has failed",
		},
		[]string{"task_name", "task_queue"},
	)

	// TaskSkippedTotal is a metric, which gets incremented each time a task
	// has failed and will be skipped from being retried.
	TaskSkippedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "task_skipped_total",
			Help:      "Total number of times a task has been skipped from being retried",
		},
		[]string{"task_name", "task_queue"},
	)

	// TaskDurationSeconds is a metric, which tracks the duration of task
	// execution in seconds.
	TaskDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "task_duration_seconds",
			Help:      "Duration of task execution in seconds",
			Buckets:   []float64{1.0, 10.0, 30.0, 60.0, 120.0},
		},
		[]string{"task_name", "task_queue"},
	)
)

// NewServer returns a new [http.Server] which can serve the metrics from
// [DefaultRegistry] on the specified network address and HTTP path. Callers
// are responsible for starting up and shutting down the HTTP server.
func NewServer(ctx context.Context, addr, path string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(
		path,
		promhttp.HandlerFor(DefaultRegistry, promhttp.HandlerOpts{}),
	)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: time.Second * 30,
		Handler:           mux,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	return server
}

// init registers collectors with the [DefaultRegistry].
func init() {
	DefaultRegistry.MustRegister(
		// Inventory metrics
		TaskSuccessfulTotal,
		TaskFailedTotal,
		TaskSkippedTotal,
		TaskDurationSeconds,
		DefaultCollector,

		// Standard Go metrics
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
}
