// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// serversDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Servers
	serversDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_servers"),
		"A gauge which tracks the number of collected OpenStack Servers",
		[]string{"project", "domain", "region"},
		nil,
	)

	// networksDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Networks
	networksDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_networks"),
		"A gauge which tracks the number of collected OpenStack Networks",
		[]string{"project", "domain", "region"},
		nil,
	)

	// subnetsDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Subnets
	subnetsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_subnets"),
		"A gauge which tracks the number of collected OpenStack Subnets",
		[]string{"project", "domain", "region"},
		nil,
	)

	// loadbalancersDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Loadbalancers
	loadbalancersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_loadbalancers"),
		"A gauge which tracks the number of collected OpenStack Loadbalancers",
		[]string{"project", "domain", "region"},
		nil,
	)

	// projectsDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Projects
	projectsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_projects"),
		"A gauge which tracks the number of collected OpenStack Projects",
		[]string{"project", "domain", "region"},
		nil,
	)

	// floatingIPsDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Floating IPs
	floatingIPsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_floating_ips"),
		"A gauge which tracks the number of collected OpenStack Floating IPs",
		[]string{"project", "domain", "region"},
		nil,
	)

	// portsDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Ports
	portsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_ports"),
		"A gauge which tracks the number of collected OpenStack Ports",
		[]string{"project", "domain", "region"},
		nil,
	)

	// routersDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Routers
	routersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_routers"),
		"A gauge which tracks the number of collected OpenStack Routers",
		[]string{"project", "domain", "region"},
		nil,
	)

	// objectsDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Objects
	objectsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_objects"),
		"A gauge which tracks the number of collected OpenStack Objects",
		[]string{"project", "domain", "region"},
		nil,
	)

	// poolsDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Pools
	poolsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_pools"),
		"A gauge which tracks the number of collected OpenStack Pools",
		[]string{"project", "domain", "region"},
		nil,
	)

	// poolMembersDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Pool Members
	poolMembersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_pool_members"),
		"A gauge which tracks the number of collected OpenStack Pool Members",
		[]string{"project", "domain", "region", "pool_id", "pool_name"},
		nil,
	)

	// containersDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack Containers
	containersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_containers"),
		"A gauge which tracks the number of collected OpenStack Containers",
		[]string{"project", "domain", "region"},
		nil,
	)

	// volumesDesc is the descriptor for a metric,
	// which tracks the number of collected OpenStack volumes
	volumesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "openstack_volumes"),
		"A gauge which tracks the number of collected OpenStack Volumes",
		[]string{"project", "domain", "region"},
		nil,
	)
)

func init() {
	metrics.DefaultCollector.AddDesc(
		serversDesc,
		networksDesc,
		subnetsDesc,
		loadbalancersDesc,
		projectsDesc,
		floatingIPsDesc,
		portsDesc,
		routersDesc,
		objectsDesc,
		poolsDesc,
		poolMembersDesc,
		containersDesc,
		volumesDesc,
	)
}
