// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"

	"github.com/gardener/inventory/pkg/aws/constants"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_IMAGES_TYPE        = "aws:task:collect-images"
	AWS_COLLECT_IMAGES_REGION_TYPE = "aws:task:collect-images-region"
)

// ErrMissingOwners is returned when expected owner names are missing.
var ErrMissingOwners = errors.New("missing owner names")

type CollectImagesPayload struct {
	ImageOwners []string `yaml:"image_owners"`
}

type CollectImagesForRegionPayload struct {
	Region      string   `yaml:"region"`
	ImageOwners []string `yaml:"image_owners"`
}

// NewCollectImagesTask creates a new task for collecting AMI Images from
// all AWS Regions.
func NewCollectImagesTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_IMAGES_TYPE, nil)
}

// NewCollectImagesForRegionTask creates a new task for collecting AMI
// Images for a given AWS Region.
func NewCollectImagesForRegionTask(payload CollectImagesForRegionPayload) (*asynq.Task, error) {
	if payload.Region == "" {
		return nil, ErrMissingRegion
	}

	if len(payload.ImageOwners) == 0 {
		return nil, ErrMissingOwners
	}

	rawPayload, err := yaml.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(AWS_COLLECT_IMAGES_REGION_TYPE, rawPayload), nil
}

// HandleCollectImagesForRegionTask collects EC2 Images for a specific Region.
func HandleCollectImagesForRegionTask(ctx context.Context, t *asynq.Task) error {
	var payload CollectImagesForRegionPayload
	if err := yaml.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("yaml.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if payload.Region == "" {
		return ErrMissingRegion
	}

	if len(payload.ImageOwners) == 0 {
		return ErrMissingOwners
	}

	return collectImagesForRegion(ctx, payload)
}

func collectImagesForRegion(ctx context.Context, payload CollectImagesForRegionPayload) error {
	region := payload.Region
	slog.Info("Collecting AWS AMI ", "region", region)
	paginator := ec2.NewDescribeImagesPaginator(
		clients.EC2,
		&ec2.DescribeImagesInput{
			Owners: payload.ImageOwners,
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
				o.Region = region
			},
		)

		if err != nil {
			slog.Error("could not describe images", "region", region, "reason", err)
			return err
		}
		items = append(items, page.Images...)
	}

	images := make([]models.Image, 0, len(items))
	for _, image := range items {
		modelImage := models.Image{
			ImageID:        strings.StringFromPointer(image.ImageId),
			Name:           strings.StringFromPointer(image.Name),
			OwnerID:        strings.StringFromPointer(image.OwnerId),
			ImageType:      string(image.ImageType),
			RootDeviceType: string(image.RootDeviceType),
			Description:    strings.StringFromPointer(image.Description),
			RegionName:     region,
		}
		images = append(images, modelImage)
	}

	if len(images) == 0 {
		return nil
	}

	out, err := clients.DB.NewInsert().
		Model(&images).
		On("CONFLICT (image_id) DO UPDATE").
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
		slog.Error("could not insert images into db", "region", region, "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws images", "region", region, "count", count)

	return nil
}

// HandleCollectImagesTask collects the EC2 Images from all known regions.
func HandleCollectImagesTask(ctx context.Context, t *asynq.Task) error {
	var payload CollectImagesPayload
	if err := yaml.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("yaml.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if len(payload.ImageOwners) == 0 {
		return ErrMissingOwners
	}

	return collectImages(ctx, payload)
}

func collectImages(ctx context.Context, payload CollectImagesPayload) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := clients.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "reason", err)
		return err
	}

	for _, r := range regions {
		// Trigger Asynq task for each region
		collectImagesForRegionPayload := CollectImagesForRegionPayload{
			Region:      r.Name,
			ImageOwners: payload.ImageOwners,
		}
		imageTask, err := NewCollectImagesForRegionTask(collectImagesForRegionPayload)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := clients.Client.Enqueue(imageTask)
		if err != nil {
			slog.Error(
				"could not enqueue task",
				"type", imageTask.Type(),
				"region", r.Name,
				"reason", err,
			)
			continue
		}

		slog.Info(
			"enqueued task",
			"type", imageTask.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
		)
	}

	return nil
}
