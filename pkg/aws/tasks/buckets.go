// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_BUCKETS_TYPE        = "aws:task:collect-buckets"
	AWS_COLLECT_BUCKETS_REGION_TYPE = "aws:task:collect-buckets-region"
)

// CollectBucketsPayload is the payload used by tasks collecting bucket information per region.
// The Region field is marshalled and passed around tasks.
type CollectBucketsPayload struct {
	Region string `json:"region"`
}

// NewCollectBucketsTask creates a new task for collecting S3 buckets from
// all AWS Regions.
func NewCollectBucketsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_BUCKETS_TYPE, nil)
}

// NewCollectBucketsForRegionTask creates a new task for collecting S3
// buckets for a given AWS Region.
func NewCollectBucketsForRegionTask(region string) (*asynq.Task, error) {
	if region == "" {
		return nil, ErrMissingRegion
	}

	payload, err := json.Marshal(CollectBucketsPayload{Region: region})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(AWS_COLLECT_BUCKETS_REGION_TYPE, payload), nil
}

// HandleCollectBucketsForRegionTask collects S3 buckets for a specific Region.
func HandleCollectBucketsForRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectBucketsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return collectBucketsForRegion(ctx, p.Region)
}

func collectBucketsForRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS S3 buckets ", "region", region)

	bucketsOutput, err := clients.S3.ListBuckets(ctx,
		&s3.ListBucketsInput{},
		func(o *s3.Options) {
			o.Region = region
		},
	)

	if err != nil {
		slog.Error("could not list buckets", "region", region, "err", err)
		return err
	}

	count := len(bucketsOutput.Buckets)
	slog.Info("found buckets", "count", count, "region", region)

	if count == 0 {
		return nil
	}

	buckets := make([]models.Bucket, 0, count)

	for _, bucket := range bucketsOutput.Buckets {
		bucketModel := models.Bucket{
			Name:         strings.StringFromPointer(bucket.Name),
			CreationDate: bucket.CreationDate,
			RegionName:   region,
		}
		buckets = append(buckets, bucketModel)
	}

	_, err = clients.DB.NewInsert().
		Model(&buckets).
		// TODO: names are supposed to be globally unique. What does it mean to come across
		// a bucket with different info with the same name?
		On("CONFLICT (name) DO UPDATE").
		Set("creation_date = EXCLUDED.creation_date").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert S3 bucket into db", "region", region, "err", err)
		return err
	}

	return nil
}

// HandleCollectBucketsTask collects the S3 buckets from all known regions.
func HandleCollectBucketsTask(ctx context.Context, t *asynq.Task) error {
	return collectBuckets(ctx)
}

func collectBuckets(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := clients.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "err", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		bucketTask, err := NewCollectBucketsForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := clients.Client.Enqueue(bucketTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", bucketTask.Type(), "err", err)
			continue
		}

		slog.Info("enqueued task", "type", bucketTask.Type(), "id", info.ID, "queue", info.Queue)
	}

	return nil
}
