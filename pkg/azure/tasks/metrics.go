// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// subscriptionsDesc is the descriptor for a metric, which tracks the number
	// of collected Azure Subscriptions.
	subscriptionsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_subscriptions"),
		"A gauge which tracks the number of collected Azure Subscriptions",
		nil,
		nil,
	)

	// vpcsDesc is the descriptor for a metric, which tracks the number
	// of collected Azure VPCs.
	vpcsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_vpcs"),
		"A gauge which tracks the number of collected Azure VPCs",
		[]string{"subscription_id", "resource_group"},
		nil,
	)

	// subnetsDesc is the descriptor for a metric, which tracks the number
	// of collected Azure Subnets.
	subnetsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_subnets"),
		"A gauge which tracks the number of collected Azure Subnets",
		[]string{"subscription_id", "resource_group", "vpc_name"},
		nil,
	)

	// loadBalancersDesc is the descriptor for a metric, which tracks the number
	// of collected Azure Load Balancers.
	loadBalancersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_load_balancers"),
		"A gauge which tracks the number of collected Azure Load Balancers",
		[]string{"subscription_id", "resource_group"},
		nil,
	)

	// blobContainersDesc is the descriptor for a metric, which tracks the
	// number of collected Azure Blob Containers.
	blobContainersDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_blob_containers"),
		"A gauge which tracks the number of collected Azure Blob Containers",
		[]string{"subscription_id", "resource_group", "storage_account"},
		nil,
	)

	// resourceGroupsDesc is the descriptor for a metric, which tracks the
	// number of collected Azure Resource Groups.
	resourceGroupsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_resource_groups"),
		"A gauge which tracks the number of collected Azure Resource Groups",
		[]string{"subscription_id"},
		nil,
	)

	// publicAddressesDesc is the descriptor for a metric, which tracks the
	// number of collected Azure Public Addresses.
	publicAddressesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_public_addresses"),
		"A gauge which tracks the number of collected Azure Public Addresses",
		[]string{"subscription_id", "resource_group"},
		nil,
	)

	// storageAccountsDesc is the descriptor for a metric, which tracks the
	// number of collected Azure Storage Accounts.
	storageAccountsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_storage_accounts"),
		"A gauge which tracks the number of collected Azure Storage Accounts",
		[]string{"subscription_id", "resource_group"},
		nil,
	)

	// virtualMachinesDesc is the descriptor for a metric, which tracks the
	// number of collected Azure Virtual Machines.
	virtualMachinesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_vms"),
		"A gauge which tracks the number of collected Azure Virtual Machines",
		[]string{"subscription_id", "resource_group"},
		nil,
	)

	// networkInterfacesDesc is the descriptor for a metric, which tracks the
	// number of collected Azure Network Interfaces.
	networkInterfacesDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "az_network_interfaces"),
		"A gauge which tracks the number of collected Azure Network Interfaces",
		[]string{"subscription_id", "resource_group"},
		nil,
	)
)

// init registers the metric descriptors with the [metrics.DefaultCollector].
func init() {
	metrics.DefaultCollector.AddDesc(
		subscriptionsDesc,
		vpcsDesc,
		subnetsDesc,
		loadBalancersDesc,
		blobContainersDesc,
		resourceGroupsDesc,
		publicAddressesDesc,
		storageAccountsDesc,
		virtualMachinesDesc,
		networkInterfacesDesc,
	)
}
