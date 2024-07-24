// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	awsClients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	TaskCollectBuckets = "aws:task:collect-buckets"
)

// NewCollectBucketsTask creates a new task for collecting S3 buckets from
// all AWS Regions.
func NewCollectBucketsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectBuckets, nil)
}

func collectBuckets(ctx context.Context) error {
	slog.Info("collecting AWS S3 buckets")

	//TODO: look into more pagination options
	bucketsOutput, err := awsClients.S3.ListBuckets(ctx,
		&s3.ListBucketsInput{},
	)

	if err != nil {
		slog.Error("could not list buckets", "reason", err)
		return err
	}

	bucketCount := len(bucketsOutput.Buckets)

	if bucketCount == 0 {
		return nil
	}

	buckets := make([]models.Bucket, 0, bucketCount)

	for _, bucket := range bucketsOutput.Buckets {
		locationOutput, err := awsClients.S3.GetBucketLocation(ctx,
			&s3.GetBucketLocationInput{
				Bucket: bucket.Name,
			})
		if err != nil {
			slog.Error(
				"could not get bucket location",
				"bucket", *bucket.Name,
				"reason", err)
		}

		region := string(locationOutput.LocationConstraint)
		// Look at the LocationConstraint (field above) documentation.
		// us-east-1 returns a nil value (empty string), so I have to
		// handle separately.
		if region == "" {
			region = "us-east-1"
		}

		bucketModel := models.Bucket{
			Name:         strings.StringFromPointer(bucket.Name),
			CreationDate: bucket.CreationDate,
			RegionName:   region,
		}
		buckets = append(buckets, bucketModel)
	}

	out, err := db.DB.NewInsert().
		Model(&buckets).
		On("CONFLICT (name) DO UPDATE").
		Set("creation_date = EXCLUDED.creation_date").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert S3 buckets into db", "err", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws s3 buckets", "count", count)

	return nil
}

// HandleCollectBucketsTask collects the S3 buckets from all known regions.
func HandleCollectBucketsTask(ctx context.Context, t *asynq.Task) error {
	return collectBuckets(ctx)
}
