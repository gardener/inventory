// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/aws/models"
	awsutils "github.com/gardener/inventory/pkg/aws/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/core/registry"
	"github.com/gardener/inventory/pkg/metrics"
	"github.com/gardener/inventory/pkg/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectBuckets is the name of the task for collecting S3 Buckets.
	TaskCollectBuckets = "aws:task:collect-buckets"
)

// NewCollectBucketsTask creates a new [asynq.Task] for collecting S3 buckets,
// without specifying a payload.
func NewCollectBucketsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectBuckets, nil)
}

// CollectBucketsPayload is the payload, which is used for collecting S3
// Buckets.
type CollectBucketsPayload struct {
	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client to use for collecting.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// HandleCollectBucketsTask handles the collection of AWS S3 Buckets.
func HandleCollectBucketsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we will enqueue tasks for
	// collecting S3 buckets for all configured AWS S3 clients.
	data := t.Payload()
	if data == nil {
		return enqueueCollectBuckets(ctx)
	}

	var payload CollectBucketsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	return collectBuckets(ctx, payload)
}

// enqueueCollectBuckets enqueues tasks for collecting AWS S3 Buckets for all
// configured AWS S3 clients.
func enqueueCollectBuckets(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	if awsclients.S3Clientset.Length() == 0 {
		logger.Warn("no AWS clients found")

		return nil
	}

	queue := asynqutils.GetQueueName(ctx)
	err := awsclients.S3Clientset.Range(func(accountID string, _ *awsclients.Client[*s3.Client]) error {
		p := CollectBucketsPayload{AccountID: accountID}
		data, err := json.Marshal(p)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS buckets",
				"account_id", accountID,
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
				"account_id", accountID,
				"reason", err,
			)

			return registry.ErrContinue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"account_id", accountID,
		)

		return nil
	})

	return err
}

// collectBuckets collects the S3 buckets for the specified account in the
// payload.
func collectBuckets(ctx context.Context, payload CollectBucketsPayload) error {
	logger := asynqutils.GetLogger(ctx)
	client, ok := awsclients.S3Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	logger.Info("collecting AWS buckets", "account_id", payload.AccountID)
	result, err := client.Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		logger.Error(
			"could not list buckets",
			"account_id", payload.AccountID,
			"reason", err,
		)

		return awsutils.MaybeSkipRetry(err)
	}

	buckets := make([]models.Bucket, 0, len(result.Buckets))
	for _, bucket := range result.Buckets {
		location, err := client.Client.GetBucketLocation(
			ctx,
			&s3.GetBucketLocationInput{
				Bucket: bucket.Name,
			},
		)
		if err != nil {
			logger.Error(
				"could not get bucket location",
				"account_id", payload.AccountID,
				"bucket", ptr.StringFromPointer(bucket.Name),
				"reason", err,
			)

			continue
		}

		// According to the AWS API documentation if the
		// LocationConstraint is empty, then this means that the region
		// is `us-east-1', so we handle it specifically.
		//
		// https://docs.aws.amazon.com/general/latest/gr/rande.html#s3_region
		region := string(location.LocationConstraint)
		if region == "" {
			region = "us-east-1"
		}

		item := models.Bucket{
			Name:         ptr.StringFromPointer(bucket.Name),
			AccountID:    payload.AccountID,
			CreationDate: ptr.Value(bucket.CreationDate, time.Time{}),
			RegionName:   region,
		}
		buckets = append(buckets, item)
	}

	if len(buckets) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&buckets).
		On("CONFLICT (name, account_id) DO UPDATE").
		Set("creation_date = EXCLUDED.creation_date").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert S3 buckets into db",
			"account_id", payload.AccountID,
			"reason", err,
		)

		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated aws s3 buckets",
		"account_id", payload.AccountID,
		"count", count,
	)

	// Emit metrics by first grouping the items by region
	groups := utils.GroupBy(buckets, func(item models.Bucket) string {
		return item.RegionName
	})
	for region, items := range groups {
		metric := prometheus.MustNewConstMetric(
			bucketsDesc,
			prometheus.GaugeValue,
			float64(len(items)),
			payload.AccountID,
			region,
		)
		key := metrics.Key(TaskCollectBuckets, payload.AccountID, region)
		metrics.DefaultCollector.AddMetric(key, metric)
	}

	return nil
}
