// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/core/registry"
)

// DefaultCollector is the default [Collector] for metrics.
var DefaultCollector = NewCollector()

// Collector is an implementation of the [prometheus.Collector] interface.
//
// This custom collector addresses some shortcomings of the upstream
// [prometheus.GaugeVec] collector. Check the documentation below for more
// details.
//
// The upstream [prometheus.GaugeVec] is not suitable for metrics reported by
// Inventory tasks such as reporting number of collected resources, primarily
// because [prometheus.GaugeVec] "remembers" any previously emitted metrics.
//
// Suppose we have a task which reports the number of collected AWS EC2
// instances, partitioned by VPC. Such a task would represent the metric as a
// gauge, because the number of EC2 instances may go up and down.
//
// Example metrics might look like this when exposed:
//
//	# HELP aws_vpc_instances Number of EC2 instances in VPC.
//	# TYPE aws_vpc_instances gauge
//	aws_vpc_instances{vpc_name="vpc-1"} 42.0
//	aws_vpc_instances{vpc_name="vpc-2"} 10.0
//
// When using [prometheus.GaugeVec] these metrics will be retained and reported
// indefinitely, even if we never collect any instances from the above AWS VPCs,
// e.g. VPCs are no longer existing, because we have deleted them.
//
// In other words the [prometheus.GaugeVec] represents the last-known value of
// the metric, as opposed to the latest value.
//
// This property makes [prometheus.GaugeVec] not suitable for some of the
// Inventory tasks, which collect resources, because we want our metric to
// represent the latest value.
type Collector struct {
	mu sync.Mutex

	// descriptors provides the [prometheus.Desc] descriptors of the metrics
	// provided by the collector.
	descriptors []*prometheus.Desc

	// reg is the internal [registry.Registry] used by the collector.
	reg *registry.Registry[string, prometheus.Metric]
}

var _ prometheus.Collector = &Collector{}

// AddDesc adds the given [prometheus.Desc] to the [Collector].
func (c *Collector) AddDesc(items ...*prometheus.Desc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.descriptors = append(c.descriptors, items...)
}

// AddMetric adds the given [prometheus.Metric] to the [Collector]. The metric
// will then be exposed by the [Collector] during scraping.
//
// The `key' is an `idempotency key', which associates a given metric and its
// label values with the internal [Collector] registry.
//
// It is up to the caller to use the same `idempotency key' for the same metric
// and label values, so that duplicate metrics are not reported by the
// collector.
func (c *Collector) AddMetric(key string, metric prometheus.Metric) {
	c.reg.Overwrite(key, metric)
}

// Describe implements the [prometheus.Collector] interface.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, desc := range c.descriptors {
		ch <- desc
	}
}

// Collect implements the [prometheus.Collector] interface.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	// After a metric has been collected we make sure that we remove it from
	// the internal registry, so that no stale metric stays with us.
	keys := make([]string, 0)
	_ = c.reg.Range(func(k string, metric prometheus.Metric) error {
		keys = append(keys, k)
		ch <- metric

		return nil
	})

	for _, k := range keys {
		c.reg.Unregister(k)
	}
}

// NewCollector creates a new [Collector]
func NewCollector() *Collector {
	c := &Collector{
		descriptors: make([]*prometheus.Desc, 0),
		reg:         registry.New[string, prometheus.Metric](),
	}

	return c
}

// Key is a utility function, which derives a key from the given items. The
// derived key can be used as an `idempotency key' for metrics when adding them
// via [Collector.AddMetric].
func Key(item string, rest ...string) string {
	items := []string{item}
	items = append(items, rest...)

	return strings.Join(items, "/")
}
