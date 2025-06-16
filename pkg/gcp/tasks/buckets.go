// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"errors"

	"cloud.google.com/go/storage"
	"github.com/hibiken/asynq"
	"google.golang.org/api/iterator"

	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	gcpclients "github.com/gardener/inventory/pkg/clients/gcp"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/gcp/models"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// TaskCollectBuckets is the name of the task for collecting GCP
	// Buckets.
	TaskCollectBuckets = "gcp:task:collect-buckets"
)

// NewCollectBucketsTask creates a new [asynq.Task] task for collecting GCP
// Buckets without specifying a payload.
func NewCollectBucketsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectBuckets, nil)
}

// CollectBucketsPayload is the payload, which is used to collect GCP Buckets.
type CollectBucketsPayload struct {
	// ProjectID specifies the GCP project ID, which is associated with a
	// registered client.
	ProjectID string `json:"project_id" yaml:"project_id"`
}

// HandleCollectBucketsTask is the handler, which collects GCP Buckets.
func HandleCollectBucketsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we will enqueue tasks for
	// collecting Buckets for all configured clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectBuckets(ctx)
	}

	// Collect Buckets using the client associated with the project ID from
	// the payload.
	var payload CollectBucketsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.ProjectID == "" {
		return asynqutils.SkipRetry(ErrNoProjectID)
	}

	return collectBuckets(ctx, payload)
}

// enqueueCollectBuckets enqueues tasks for collecting GCP Buckets
// for all collected GCP projects.
func enqueueCollectBuckets(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)

	queue := asynqutils.GetQueueName(ctx)
	err := gcpclients.StorageClientset.Range(func(projectID string, _ *gcpclients.Client[*storage.Client]) error {
		p := &CollectBucketsPayload{ProjectID: projectID}
		data, err := json.Marshal(p)
		if err != nil {
			logger.Error(
				"failed to marshal payload for GCP Buckets",
				"project", projectID,
				"reason", err,
			)

			return registry.ErrContinue
		}

		task := asynq.NewTask(TaskCollectBuckets, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"project", projectID,
				"reason", err,
			)

			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"project", projectID,
		)

		return nil
	})

	return err
}

// collectBuckets collects the GCP Buckets using the client configuration
// specified in the payload.
func collectBuckets(ctx context.Context, payload CollectBucketsPayload) error {
	client, ok := gcpclients.StorageClientset.Get(payload.ProjectID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.ProjectID))
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			bucketsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.ProjectID,
		)
		key := metrics.Key(TaskCollectBuckets, payload.ProjectID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger := asynqutils.GetLogger(ctx)
	logger.Info("collecting GCP buckets", "project", payload.ProjectID)

	iter := client.Client.Buckets(ctx, payload.ProjectID)

	items := make([]models.Bucket, 0)

	for {
		b, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			logger.Error("failed to get buckets",
				"project", payload.ProjectID,
				"reason", err,
			)

			return err
		}

		item := models.Bucket{
			Name:                b.Name,
			ProjectID:           payload.ProjectID,
			LocationType:        b.LocationType,
			Location:            b.Location,
			DefaultStorageClass: b.StorageClass,
			CreationTimestamp:   b.Created.String(),
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (name, project_id) DO UPDATE").
		Set("location_type = EXCLUDED.location_type").
		Set("location = EXCLUDED.location").
		Set("default_storage_class = EXCLUDED.default_storage_class").
		Set("creation_timestamp = EXCLUDED.creation_timestamp").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert buckets into db",
			"project", payload.ProjectID,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated gcp buckets",
		"project", payload.ProjectID,
		"count", count,
	)

	return nil
}
