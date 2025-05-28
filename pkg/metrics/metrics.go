// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
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
	// TaskExecutionTotal is a metric, which gets incremented each time a
	// task has been called.
	TaskExecutionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "task_execution_total",
			Help:      "Total number of times a task has been executed",
		},
		[]string{"task_name", "task_queue"},
	)
)

// NewServer returns a new [http.Server] which can serve the metrics from
// [DefaultRegistry] on the specified network address and HTTP path. Callers
// are responsible for starting up and shutting down the HTTP server.
func NewServer(addr, path string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle(
		path,
		promhttp.HandlerFor(DefaultRegistry, promhttp.HandlerOpts{}),
	)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: time.Second * 30,
		Handler:           mux,
	}

	return server
}

// init registers collectors with the [DefaultRegistry].
func init() {
	DefaultRegistry.MustRegister(
		// Inventory metrics
		TaskExecutionTotal,

		// Standard Go metrics
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
}
