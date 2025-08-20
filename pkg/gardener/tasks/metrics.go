// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// projectsDesc is the descriptor for a metric, which tracks the number
	// of collected Gardener Projects.
	projectsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_projects"),
		"A gauge which tracks the number of collected Gardener projects",
		nil,
		nil,
	)

	// projectMembersDesc is the descriptor for a metric, which tracks the
	// number of collected Gardener Project members.
	projectMembersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_project_members"),
		"A gauge which tracks the number of collected Gardener project members",
		[]string{"project_name"},
		nil,
	)

	// shootsDesc is the descriptor for a metric, which tracks the number of
	// collected Gardener Shoots.
	shootsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_shoots"),
		"A gauge which tracks the number of collected Gardener shoots",
		[]string{"project_name"},
		nil,
	)

	// seedsDesc is the descriptor for a metric, which tracks the number
	// of collected Gardener Seeds.
	seedsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_seeds"),
		"A gauge which tracks the number of collected Gardener seeds",
		nil,
		nil,
	)

	// machinesDesc is the descriptor for a metric, which tracks the number
	// of collected Gardener Machines from seeds.
	machinesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_machines"),
		"A gauge which tracks the number of collected Gardener machines",
		[]string{"seed"},
		nil,
	)

	// backupBucketsDesc is the descriptor for a metric, which tracks the
	// number of collected Gardener Backup Buckets.
	backupBucketsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_backup_buckets"),
		"A gauge which tracks the number of collected Gardener backup buckets",
		nil,
		nil,
	)

	// cloudProfilesDesc is the descriptor for a metric, which tracks the
	// number of collected Gardener Cloud Profiles.
	cloudProfilesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_cloud_profiles"),
		"A gauge which tracks the number of collected Gardener Cloud Profiles",
		nil,
		nil,
	)

	// seedVolumesDesc is the descriptor for a metric, which tracks the
	// number of collected Persitent Volumes from seed clusters.
	seedVolumesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_seed_volumes"),
		"A gauge which tracks the number of collected persistent volumes from seeds",
		[]string{"seed"},
		nil,
	)

	// dnsRecordsDesc is the descriptor for a metric, which tracks the
	// number of collected Gardener DNSRecords from seed clusters.
	dnsRecordsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_dns_records"),
		"A gauge which tracks the number of collected Gardener DNSRecords from seeds",
		[]string{"seed"},
		nil,
	)

	// dnsEntriesDesc is the descriptor for a metric, which tracks the
	// number of collected Gardener DNSEntry resources from seed clusters.
	dnsEntriesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "g_dns_entries"),
		`A gauge which tracks the number of collected Gardener DNSEntry
		resources from seeds`,
		[]string{"seed"},
		nil,
	)
)

// init registers metrics with the [metrics.DefaultCollector].
func init() {
	metrics.DefaultCollector.AddDesc(
		projectsDesc,
		projectMembersDesc,
		shootsDesc,
		seedsDesc,
		machinesDesc,
		backupBucketsDesc,
		cloudProfilesDesc,
		seedVolumesDesc,
		dnsRecordsDesc,
		dnsEntriesDesc,
	)
}
