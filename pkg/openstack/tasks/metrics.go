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
)

func init() {
	metrics.DefaultCollector.AddDesc(
		serversDesc,
	)
}
