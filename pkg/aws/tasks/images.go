package tasks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gopkg.in/yaml.v3"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	awsclients "github.com/gardener/inventory/pkg/aws/clients"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/aws/utils"
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
	ImageOwners []int64 `yaml:"image_owners"`
}

type CollectImagesForRegionPayload struct {
	Region      string  `yaml:"region"`
	ImageOwners []int64 `yaml:"image_owners"`
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

	if payload.ImageOwners == nil || len(payload.ImageOwners) == 0 {
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

	if payload.ImageOwners == nil || len(payload.ImageOwners) == 0 {
		return ErrMissingOwners
	}

	return collectImagesForRegion(ctx, payload)
}

func collectImagesForRegion(ctx context.Context, payload CollectImagesForRegionPayload) error {
	region := payload.Region

	slog.Info("Collecting AWS AMI ", "region", region)

	owners := make([]string, 0, len(payload.ImageOwners))
	for _, o := range payload.ImageOwners {
		owners = append(owners, fmt.Sprintf("%d", o))
	}

	imagesOutput, err := awsclients.Ec2.DescribeImages(ctx,
		&ec2.DescribeImagesInput{
			Owners: owners,
		},
		func(o *ec2.Options) {
			o.Region = region
		},
	)

	if err != nil {
		slog.Error("could not describe images", "err", err)
		return err
	}

	// Parse reservations and add to images
	count := len(imagesOutput.Images)
	slog.Info("found images", "count", count, "region", region)

	images := make([]models.Image, 0, count)

	for _, image := range imagesOutput.Images {
		name := utils.FetchTag(image.Tags, "Name")
		modelImage := models.Image{
			ImageID:        strings.StringFromPointer(image.ImageId),
			Name:           name,
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

	_, err = clients.DB.NewInsert().
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
		slog.Error("could not insert images into db", "err", err)
		return err
	}

	return nil
}

// HandleCollectImagesTask collects the EC2 Images from all known regions.
func HandleCollectImagesTask(ctx context.Context, t *asynq.Task) error {
	var payload CollectImagesPayload
	if err := yaml.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("yaml.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if payload.ImageOwners == nil || len(payload.ImageOwners) == 0 {
		return ErrMissingOwners
	}

	return collectImages(ctx, payload)
}

func collectImages(ctx context.Context, payload CollectImagesPayload) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := clients.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "err", err)
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
			slog.Error("could not enqueue task", "type", imageTask.Type(), "err", err)
			continue
		}

		slog.Info("enqueued task", "type", imageTask.Type(), "id", info.ID, "queue", info.Queue)
	}

	return nil
}
