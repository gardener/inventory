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
	// of collected GCP projects.
	projectsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_projects"),
		"A gauge which tracks the number of collected GCP projects",
		nil,
		nil,
	)

	// vpcsDesc is the descriptor for a metric, which tracks the number
	// of collected GCP VPC networks.
	vpcsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_vpcs"),
		"A gauge which tracks the number of collected GCP VPCs",
		[]string{"project_id"},
		nil,
	)

	// disksDesc is the descriptor for a metric, which tracks the number
	// of collected GCP disks.
	disksDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_disks"),
		"A gauge which tracks the number of collected GCP disks",
		[]string{"project_id"},
		nil,
	)

	// bucketsDesc is the descriptor for a metric, which tracks the number
	// of collected GCP buckets.
	bucketsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_buckets"),
		"A gauge which tracks the number of collected GCP buckets",
		[]string{"project_id"},
		nil,
	)

	// subnetsDesc is the descriptor for a metric, which tracks the number
	// of collected GCP subnets.
	subnetsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_subnets"),
		"A gauge which tracks the number of collected GCP subnets",
		[]string{"project_id"},
		nil,
	)

	// addressesDesc is the descriptor for a metric, which tracks the number
	// of collected GCP regional and global addresses.
	addressesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_addresses"),
		"A gauge which tracks the number of collected GCP addresses",
		[]string{"project_id"},
		nil,
	)

	// instancesDesc is the descriptor for a metric, which tracks the number
	// of collected GCP instances.
	instancesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_instances"),
		"A gauge which tracks the number of collected GCP instances",
		[]string{"project_id"},
		nil,
	)

	// gkeClustersDesc is the descriptor for a metric, which tracks the number
	// of collected GKE clusters.
	gkeClustersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_gke_clusters"),
		"A gauge which tracks the number of collected GKE clusters",
		[]string{"project_id"},
		nil,
	)

	// targetPoolsDesc is the descriptor for a metric, which tracks the number
	// of collected GCP target pools.
	targetPoolsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_target_pools"),
		"A gauge which tracks the number of collected GCP target pools",
		[]string{"project_id"},
		nil,
	)

	// forwardingRulesDesc is the descriptor for a metric, which tracks
	// the number of collected GCP Forwarding Rules.
	forwardingRulesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_forwarding_rules"),
		"A gauge which tracks the number of collected GCP forwarding rules",
		[]string{"project_id"},
		nil,
	)

	// iamPoliciesDesc is the descriptor for a metric, which tracks the number
	// of collected GCP IAM policies.
	iamPoliciesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "gcp_iam_policies"),
		"A gauge which tracks the number of collected GCP IAM policies",
		[]string{"project_id"},
		nil,
	)
)

// init registers the metrics with the [metrics.DefaultCollector].
func init() {
	metrics.DefaultCollector.AddDesc(
		projectsDesc,
		vpcsDesc,
		subnetsDesc,
		disksDesc,
		bucketsDesc,
		addressesDesc,
		instancesDesc,
		gkeClustersDesc,
		targetPoolsDesc,
		forwardingRulesDesc,
		iamPoliciesDesc,
	)
}
