// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// CollectedProjectsMetric is a gauge, which tracks the number of
	// collected Gardener Projects.
	CollectedProjectsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_projects",
			Help:      "A gauge which tracks the number of collected Gardener projects",
		},
	)

	// CollectedProjectMembersMetric is a gauge, which tracks the number of
	// collected Gardener Project members.
	CollectedProjectMembersMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_project_members",
			Help:      "A gauge which tracks the number of collected Gardener project members",
		},
		[]string{"project_name"},
	)

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
		CollectedProjectsMetric,
		CollectedProjectMembersMetric,
		CollectedShootsMetric,
	)
}
