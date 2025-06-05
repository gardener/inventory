// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// collectedProjectsMetric is a gauge, which tracks the number of
	// collected Gardener Projects.
	collectedProjectsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_projects",
			Help:      "A gauge which tracks the number of collected Gardener projects",
		},
	)

	// collectedProjectMembersMetric is a gauge, which tracks the number of
	// collected Gardener Project members.
	collectedProjectMembersMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_project_members",
			Help:      "A gauge which tracks the number of collected Gardener project members",
		},
		[]string{"project_name"},
	)

	// collectedShootsMetric is a gauge, which tracks the number of
	// collected Gardener Shoots.
	collectedShootsMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_shoots",
			Help:      "A gauge which tracks the number of collected Gardener shoots",
		},
		[]string{"project_name"},
	)

	// collectedSeedsMetric is a gauge, which tracks the number of
	// collected Gardener Seeds.
	collectedSeedsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_seeds",
			Help:      "A gauge which tracks the number of collected Gardener seeds",
		},
	)

	// collectedMachinesMetric is a gauge, which tracks the number of
	// collected Gardener Machines from seeds.
	collectedMachinesMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_machines",
			Help:      "A gauge which tracks the number of collected Gardener machines",
		},
		[]string{"seed"},
	)

	// collectedBackupBucketsMetric is a gauge, which tracks the number of
	// collected Gardener Backup Buckets.
	collectedBackupBucketsMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_backup_buckets",
			Help:      "A gauge which tracks the number of collected Gardener backup buckets",
		},
	)

	// collectedCloudProfilesMetric is a gauge, which tracks the number of
	// collected Gardener Cloud Profiles.
	collectedCloudProfilesMetric = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_cloud_profiles",
			Help:      "A gauge which tracks the number of collected Gardener Cloud Profiles",
		},
	)

	// collectedSeedVolumesMetric is a gauge, which tracks the number of
	// collected Persitent Volumes from seed clusters.
	collectedSeedVolumesMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metrics.Namespace,
			Name:      "g_collected_seed_volumes",
			Help:      "A gauge which tracks the number of collected persistent volumes from seeds",
		},
		[]string{"seed"},
	)
)

// init registers metrics with the [metrics.DefaultRegistry].
func init() {
	metrics.DefaultRegistry.MustRegister(
		collectedProjectsMetric,
		collectedProjectMembersMetric,
		collectedShootsMetric,
		collectedSeedsMetric,
		collectedMachinesMetric,
		collectedBackupBucketsMetric,
		collectedCloudProfilesMetric,
		collectedSeedVolumesMetric,
	)
}
