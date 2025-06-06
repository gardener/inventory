// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// projectsMetric is a gauge, which tracks the number of
	// collected Gardener Projects.
	projectsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_projects",
			Help:      "A gauge which tracks the number of collected Gardener projects",
		},
	)

	// projectMembersMetric is a gauge, which tracks the number of
	// collected Gardener Project members.
	projectMembersMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_project_members",
			Help:      "A gauge which tracks the number of collected Gardener project members",
		},
		[]string{"project_name"},
	)

	// shootsMetric is a gauge, which tracks the number of
	// collected Gardener Shoots.
	shootsMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_shoots",
			Help:      "A gauge which tracks the number of collected Gardener shoots",
		},
		[]string{"project_name"},
	)

	// seedsMetric is a gauge, which tracks the number of
	// collected Gardener Seeds.
	seedsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_seeds",
			Help:      "A gauge which tracks the number of collected Gardener seeds",
		},
	)

	// machinesMetric is a gauge, which tracks the number of
	// collected Gardener Machines from seeds.
	machinesMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_machines",
			Help:      "A gauge which tracks the number of collected Gardener machines",
		},
		[]string{"seed"},
	)

	// backupBucketsMetric is a gauge, which tracks the number of
	// collected Gardener Backup Buckets.
	backupBucketsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_backup_buckets",
			Help:      "A gauge which tracks the number of collected Gardener backup buckets",
		},
	)

	// cloudProfilesMetric is a gauge, which tracks the number of
	// collected Gardener Cloud Profiles.
	cloudProfilesMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_cloud_profiles",
			Help:      "A gauge which tracks the number of collected Gardener Cloud Profiles",
		},
	)

	// seedVolumesMetric is a gauge, which tracks the number of
	// collected Persitent Volumes from seed clusters.
	seedVolumesMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_seed_volumes",
			Help:      "A gauge which tracks the number of collected persistent volumes from seeds",
		},
		[]string{"seed"},
	)
)

// init registers metrics with the [metrics.DefaultRegistry].
func init() {
	metrics.DefaultRegistry.MustRegister(
		projectsMetric,
		projectMembersMetric,
		shootsMetric,
		seedsMetric,
		machinesMetric,
		backupBucketsMetric,
		cloudProfilesMetric,
		seedVolumesMetric,
	)
}
