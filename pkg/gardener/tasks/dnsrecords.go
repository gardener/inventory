// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gardenerclient "github.com/gardener/inventory/pkg/clients/gardener"
	"github.com/gardener/inventory/pkg/gardener/models"
	gutils "github.com/gardener/inventory/pkg/gardener/utils"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectDNSRecords is the name of the task for collecting Gardener
	// DNSRecords.
	TaskCollectDNSRecords = "g:task:collect-dns-records"
)

// CollectDNSRecordsPayload is the payload, which is used for collecting Gardener
// DNSRecords.
type CollectDNSRecordsPayload struct {
	// Seed is the name of the seed cluster from which to collect Gardener
	// DNSRecords.
	Seed string `json:"seed" yaml:"seed"`
}

// NewCollectDNSRecordsTask creates a new [asynq.Task] for collecting Gardener
// DNSRecords, without specifying a payload.
func NewCollectDNSRecordsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectDNSRecords, nil)
}

// HandleCollectDNSRecordsTask is the handler for collecting Gardener DNSRecords.
func HandleCollectDNSRecordsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting DNSRecords from all known Gardener Seed clusters.
	data := t.Payload()
	if data == nil {
		return enqueueCollectDNSRecords(ctx)
	}

	var payload CollectDNSRecordsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Seed == "" {
		return asynqutils.SkipRetry(ErrNoSeedCluster)
	}

	return collectDNSRecords(ctx, payload)
}

// enqueueCollectDNSRecords enqueues tasks for collecting Gardener DNSRecords from
// all known Seed Clusters.
func enqueueCollectDNSRecords(ctx context.Context) error {
	seeds, err := gutils.GetSeedsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seeds from db: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Create a task for each known seed cluster
	for _, s := range seeds {
		payload := CollectDNSRecordsPayload{
			Seed: s.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Gardener DNS records",
				"seed", s.Name,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectDNSRecords, data)
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

	return nil
}

// collectDNSRecords collects the Gardener DNSRecords from the Seed Cluster
// specified in the payload.
func collectDNSRecords(ctx context.Context, payload CollectDNSRecordsPayload) error {
	logger := asynqutils.GetLogger(ctx)
	if !gardenerclient.IsDefaultClientSet() {
		logger.Warn("gardener client not configured")

		return nil
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			dnsRecordsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.Seed,
		)
		key := metrics.Key(TaskCollectDNSRecords, payload.Seed)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger.Info("collecting Gardener DNS records", "seed", payload.Seed)
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

	scheme := runtime.NewScheme()
	err = extensionsv1alpha1.AddToScheme(scheme)
	if err != nil {
		return asynqutils.SkipRetry(fmt.Errorf("could not add DNS record scheme to client for seed %q: %s", payload.Seed, err))
	}

	client, err := crtclient.New(restConfig, crtclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		return asynqutils.SkipRetry(fmt.Errorf("cannot create client for seed %q: %s", payload.Seed, err))
	}

	var result extensionsv1alpha1.DNSRecordList

	err = client.List(ctx, &result)
	if err != nil {
		return fmt.Errorf("cannot list DNS records for seed %q: %s", payload.Seed, err)
	}

	dnsRecords := make([]models.DNSRecord, 0)

	for _, item := range result.Items {
		spec := item.Spec

		name := item.Name
		namespace := item.Namespace
		fqdn := spec.Name
		recordType := string(spec.RecordType)

		ttl := spec.TTL

		region := ptr.StringFromPointer(spec.Region)
		dnsZone := ptr.StringFromPointer(spec.Zone)

		creationTimestamp := item.CreationTimestamp.Time

		for _, value := range spec.Values {
			record := models.DNSRecord{
				Name:              name,
				Namespace:         namespace,
				FQDN:              fqdn,
				RecordType:        recordType,
				ProviderType:      spec.Type,
				Value:             value,
				TTL:               ttl,
				Region:            region,
				DNSZone:           dnsZone,
				SeedName:          payload.Seed,
				CreationTimestamp: creationTimestamp,
			}
			dnsRecords = append(dnsRecords, record)
		}
	}

	if err != nil {
		return fmt.Errorf("could not list DNS records for seed %q: %w", payload.Seed, err)
	}

	if len(dnsRecords) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&dnsRecords).
		On("CONFLICT (name, namespace, seed_name, value) DO UPDATE").
		Set("fqdn = EXCLUDED.fqdn").
		Set("record_type = EXCLUDED.record_type").
		Set("provider_type = EXCLUDED.provider_type").
		Set("ttl = EXCLUDED.ttl").
		Set("region = EXCLUDED.region").
		Set("dns_zone = EXCLUDED.dns_zone").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert Gardener DNS records into db",
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
		"populated Gardener DNS records",
		"seed", payload.Seed,
		"count", count,
	)

	return nil
}
