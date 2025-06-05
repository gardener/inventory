// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// CollectedShootsMetric is a gauge, which tracks the number of
	// collected Gardener Shoots.
	CollectedShootsMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_shoots",
			Help:      "A gauge which tracks the number of collected Gardener shoots",
		},
		[]string{"project_name"},
	)
)

// init registers metrics with the [metrics.DefaultRegistry].
func init() {
	metrics.DefaultRegistry.MustRegister(
		CollectedShootsMetric,
	)
}
