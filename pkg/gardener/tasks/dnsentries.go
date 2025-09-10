// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	dnsapi "github.com/gardener/external-dns-management/pkg/apis/dns/v1alpha1"
	dnsclientset "github.com/gardener/external-dns-management/pkg/client/dns/clientset/versioned"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/pager"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/constants"
	"github.com/gardener/inventory/pkg/gardener/models"
	gutils "github.com/gardener/inventory/pkg/gardener/utils"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectDNSEntries is the name of the task for collecting Gardener
	// DNSEntry resources.
	TaskCollectDNSEntries = "g:task:collect-dns-entries"

	gardenClusterIdentifier = "garden"
)

// CollectDNSEntriesPayload is the payload, which is used for collecting Gardener
// DNSEntry resources.
type CollectDNSEntriesPayload struct {
	// Seed is the name of the seed cluster from which to collect Gardener
	// DNSEntry resources.
	Seed string `json:"seed" yaml:"seed"`

	// TargetGarden is the flag responsible for collecting from the garden
	// cluster instead of a seed.
	TargetGarden bool `json:"target_garden" yaml:"target_garden"`
}

// NewCollectDNSEntriesTask creates a new [asynq.Task] for collecting Gardener
// DNSEntry resources, without specifying a payload.
func NewCollectDNSEntriesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectDNSEntries, nil)
}

// HandleCollectDNSEntriesTask is the handler for collecting Gardener DNSEntry
// resources.
func HandleCollectDNSEntriesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting DNSEntry resources from the Garden cluster and all known
	// Gardener Seed clusters.
	data := t.Payload()
	if data == nil {
		return enqueueCollectDNSEntries(ctx)
	}

	var payload CollectDNSEntriesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.TargetGarden {
		return collectDNSEntries(ctx, payload)
	}

	if payload.Seed == "" {
		return asynqutils.SkipRetry(ErrNoSeedCluster)
	}

	return collectDNSEntries(ctx, payload)
}

// enqueueCollectDNSEntries enqueues tasks for collecting Gardener DNSentry
// resources from all known Seed Clusters.
func enqueueCollectDNSEntries(ctx context.Context) error {
	seeds, err := gutils.GetSeedsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seeds from db: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	for _, s := range seeds {
		payload := CollectDNSEntriesPayload{
			Seed: s.Name,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Gardener DNS entries",
				"seed", s.Name,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectDNSEntries, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"seed", s.Name,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"seed", s.Name,
		)
	}

	// enqueue task for Garden cluster
	payload := CollectDNSEntriesPayload{
		Seed:         "",
		TargetGarden: true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		logger.Error(
			"failed to marshal payload for DNS entries for the garden cluster",
			"reason", err,
		)

		return err
	}

	task := asynq.NewTask(TaskCollectDNSEntries, data)
	info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
	if err != nil {
		logger.Error(
			"failed to enqueue task for garden cluster",
			"type", task.Type(),
			"reason", err,
		)

		return err
	}

	logger.Info(
		"enqueued task for garden cluster",
		"type", task.Type(),
		"id", info.ID,
		"queue", info.Queue,
	)

	return nil
}

// collectDNSEntries collects the Gardener DNSentry resources from the Seed Cluster
// specified in the payload.
func collectDNSEntries(ctx context.Context, payload CollectDNSEntriesPayload) error {
	if payload.TargetGarden {
		return collectDNSEntriesFromGarden(ctx, payload)
	}

	return collectDNSEntriesFromSeed(ctx, payload)
}

// collectDNSEntriesFromSeed collects the Gardener DNSentry resources from the Seed Cluster
// specified in the payload.
func collectDNSEntriesFromSeed(ctx context.Context, payload CollectDNSEntriesPayload) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			dnsEntriesDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Seed,
		)
		key := metrics.Key(TaskCollectDNSEntries, payload.Seed)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger.Info("collecting Gardener DNS entries", "seed", payload.Seed)
	restConfig, err := gardenerclient.DefaultClient.SeedRestConfig(ctx, payload.Seed)
	if err != nil {
		if errors.Is(err, gardenerclient.ErrSeedIsExcluded) {
			// Don't treat excluded seeds as errors, in order to
			// avoid accumulating archived tasks
			logger.Warn("seed is excluded", "seed", payload.Seed)

			return nil
		}

		return asynqutils.SkipRetry(fmt.Errorf("cannot get rest config for seed %q: %s", payload.Seed, err))
	}

	client, err := dnsclientset.NewForConfig(restConfig)
	if err != nil {
		return asynqutils.SkipRetry(fmt.Errorf("cannot create client for dns entries %q: %s", payload.Seed, err))
	}

	dnsEntries := make([]models.DNSEntry, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.DnsV1alpha1().DNSEntries("").List(ctx, opts)
		}),
	)

	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		entry, ok := obj.(*dnsapi.DNSEntry)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		name := entry.Name
		namespace := entry.Namespace
		fqdn := entry.Spec.DNSName

		// combine Spec.Targets and Spec.Text, as either one or the other
		// can be specified
		values := entry.Spec.Targets
		values = append(values, entry.Spec.Text...)

		ttl := entry.Spec.TTL

		dnsZone := ptr.StringFromPointer(entry.Status.Zone)

		providerType := ptr.StringFromPointer(entry.Status.ProviderType)
		provider := ptr.StringFromPointer(entry.Status.Provider)

		creationTimestamp := entry.CreationTimestamp.Time

		for _, value := range values {
			item := models.DNSEntry{
				Name:              name,
				Namespace:         namespace,
				FQDN:              fqdn,
				Value:             value,
				TTL:               ttl,
				DNSZone:           dnsZone,
				ProviderType:      providerType,
				Provider:          provider,
				SeedName:          payload.Seed,
				CreationTimestamp: creationTimestamp,
			}
			dnsEntries = append(dnsEntries, item)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list dns entries for seed %q: %w", payload.Seed, err)
	}

	if len(dnsEntries) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&dnsEntries).
		On("CONFLICT (name, namespace, seed_name, value) DO UPDATE").
		Set("fqdn = EXCLUDED.fqdn").
		Set("value = EXCLUDED.value").
		Set("ttl = EXCLUDED.ttl").
		Set("dns_zone = EXCLUDED.dns_zone").
		Set("provider_type = EXCLUDED.provider_type").
		Set("provider = EXCLUDED.provider").
		Set("seed_name = EXCLUDED.seed_name").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert Gardener DNS entries into db",
			"seed", payload.Seed,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated Gardener DNS entries",
		"seed", payload.Seed,
		"count", count,
	)

	return nil
}

// collectDNSEntriesFromGarden collects the Gardener DNSentry resources from the Garden Cluster
func collectDNSEntriesFromGarden(ctx context.Context, payload CollectDNSEntriesPayload) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			dnsEntriesDesc,
			prometheus.GaugeValue,
			float64(count),
			gardenClusterIdentifier,
		)
		key := metrics.Key(TaskCollectDNSEntries, gardenClusterIdentifier)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger.Info("collecting DNS entries for garden cluster")
	gardenRestConfig := gardenerclient.DefaultClient.RESTConfig()

	client, err := dnsclientset.NewForConfig(gardenRestConfig)
	if err != nil {
		logger.Error(
			"could not create garden client",
			"seed", gardenClusterIdentifier,
			"reason", err,
		)

		return err
	}

	dnsEntries := make([]models.DNSEntry, 0)
	p := pager.New(
		pager.SimplePageFunc(func(opts metav1.ListOptions) (runtime.Object, error) {
			return client.DnsV1alpha1().DNSEntries("").List(ctx, opts)
		}),
	)

	opts := metav1.ListOptions{Limit: constants.PageSize}
	err = p.EachListItem(ctx, opts, func(obj runtime.Object) error {
		entry, ok := obj.(*dnsapi.DNSEntry)
		if !ok {
			return fmt.Errorf("unexpected object type: %T", obj)
		}

		name := entry.Name
		namespace := entry.Namespace
		fqdn := entry.Spec.DNSName

		// combine Spec.Targets and Spec.Text, as either one or the other
		// can be specified
		values := entry.Spec.Targets
		values = append(values, entry.Spec.Text...)

		ttl := entry.Spec.TTL

		dnsZone := ptr.StringFromPointer(entry.Status.Zone)

		providerType := ptr.StringFromPointer(entry.Status.ProviderType)
		provider := ptr.StringFromPointer(entry.Status.Provider)

		creationTimestamp := entry.CreationTimestamp.Time

		for _, value := range values {
			item := models.DNSEntry{
				Name:              name,
				Namespace:         namespace,
				FQDN:              fqdn,
				Value:             value,
				TTL:               ttl,
				DNSZone:           dnsZone,
				ProviderType:      providerType,
				Provider:          provider,
				SeedName:          gardenClusterIdentifier,
				CreationTimestamp: creationTimestamp,
			}
			dnsEntries = append(dnsEntries, item)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("could not list dns entries for seed %q: %w", payload.Seed, err)
	}

	if len(dnsEntries) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&dnsEntries).
		On("CONFLICT (name, namespace, seed_name, value) DO UPDATE").
		Set("fqdn = EXCLUDED.fqdn").
		Set("ttl = EXCLUDED.ttl").
		Set("dns_zone = EXCLUDED.dns_zone").
		Set("provider_type = EXCLUDED.provider_type").
		Set("provider = EXCLUDED.provider").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert Gardener DNS entries into db",
			"seed", payload.Seed,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated Gardener DNS entries",
		"seed", payload.Seed,
		"count", count,
	)

	return nil
}
