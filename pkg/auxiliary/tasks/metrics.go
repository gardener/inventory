// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/metrics"
)

var (
	// hkDeletedRecordsDesc is the descriptor for a metric, which tracks the
	// number of deleted resources for models by the housekeeper.
	hkDeletedRecordsDesc = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.Namespace, "", "housekeeper_deleted_records"),
		"Gauge which tracks the number of deleted records by the housekeeper",
		[]string{"model_name"},
		nil,
	)
)

// init registers the metric descriptors with the [metrics.DefaultCollector]
func init() {
	metrics.DefaultCollector.AddDesc(
		hkDeletedRecordsDesc,
	)
}
