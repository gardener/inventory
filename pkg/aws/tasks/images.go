// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/aws/constants"
	"github.com/gardener/inventory/pkg/aws/models"
	awsutils "github.com/gardener/inventory/pkg/aws/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectImages is the name of the task for collecting AWS AMIs.
	TaskCollectImages = "aws:task:collect-images"
)

// CollectImagesPayload is the payload, which is used for collecting AWS AMIs.
type CollectImagesPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`

	// Owners specifies owners of AMI images. Only images with the specified
	// owners will be collected.
	Owners []string `json:"owners" yaml:"owners"`
}

// NewCollectImagesTask creates a new [asynq.Task] for collecting AWS AMIs
// without specifying a payload.
func NewCollectImagesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectImages, nil)
}

// HandleCollectImagesTask handles the task for collecting AWS AMIs.
func HandleCollectImagesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a specified region and account id, then we
	// will enqueue tasks for collecting AWS AMIs from all known regions.
	data := t.Payload()
	var payload CollectImagesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	switch {
	case payload.Region == "":
		// We don't have a specified region, enqueue collection for all
		// known regions.
		return enqueueCollectImages(ctx, payload)
	case payload.AccountID == "":
		// Required AccountID is missing
		return asynqutils.SkipRetry(ErrNoAccountID)
	default:
		// Collect AMIs
		return collectImages(ctx, payload)
	}
}

// enqueueCollectImages enqueues tasks for collecting AWS AMIs from all known
// regions and accounts.
func enqueueCollectImages(ctx context.Context, payload CollectImagesPayload) error {
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)
	// Enqueue task for each known region
	for _, r := range regions {
		if !awsclients.EC2Clientset.Exists(r.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", r.Name,
				"account_id", r.AccountID,
			)

			continue
		}

		// By default we will specify the current account id as the
		// image owner, unless specified as part of the payload.
		owners := payload.Owners
		if len(owners) == 0 {
			owners = []string{r.AccountID}
		}
		payload := CollectImagesPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
			Owners:    owners,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS AMIs",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectImages, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
			"account_id", r.AccountID,
		)
	}

	return nil
}

// collectImages collects the AWS AMIs based on the specified payload.
func collectImages(ctx context.Context, payload CollectImagesPayload) error {
	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			imagesDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.AccountID,
			payload.Region,
		)
		key := metrics.Key(TaskCollectImages, payload.AccountID, payload.Region)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger := asynqutils.GetLogger(ctx)
	paginator := ec2.NewDescribeImagesPaginator(
		client.Client,
		&ec2.DescribeImagesInput{
			Owners: payload.Owners,
		},
		func(params *ec2.DescribeImagesPaginatorOptions) {
			params.Limit = int32(constants.PageSize)
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.Image, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *ec2.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not describe AMIs",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return awsutils.MaybeSkipRetry(err)
		}

		items = append(items, page.Images...)
	}

	images := make([]models.Image, 0, len(items))
	for _, image := range items {
		item := models.Image{
			ImageID:        ptr.StringFromPointer(image.ImageId),
			AccountID:      payload.AccountID,
			Name:           ptr.StringFromPointer(image.Name),
			OwnerID:        ptr.StringFromPointer(image.OwnerId),
			ImageType:      string(image.ImageType),
			RootDeviceType: string(image.RootDeviceType),
			Description:    ptr.StringFromPointer(image.Description),
			RegionName:     payload.Region,
		}
		images = append(images, item)
	}

	if len(images) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&images).
		On("CONFLICT (image_id, account_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("owner_id = EXCLUDED.owner_id").
		Set("image_type = EXCLUDED.image_type").
		Set("root_device_type = EXCLUDED.root_device_type").
		Set("description = EXCLUDED.description").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert AMIs into db",
			"region", payload.Region,
			"account_id", payload.AccountID,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated aws amis",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}
