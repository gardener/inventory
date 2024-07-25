// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/constants"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/aws/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclient "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	TaskCollectInstances          = "aws:task:collect-instances"
	TaskCollectInstancesForRegion = "aws:task:collect-instances-region"
)

type CollectInstancesPayload struct {
	Region string `json:"region"`
}

// NewCollectInstancesTask creates a new task for collecting EC2 Instances from
// all AWS Regions.
func NewCollectInstancesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectInstances, nil)
}

// NewCollectInstancesForRegionTask creates a new task for collecting EC2
// Instances for a given AWS Region.
func NewCollectInstancesForRegionTask(region string) (*asynq.Task, error) {
	if region == "" {
		return nil, ErrMissingRegion
	}

	payload, err := json.Marshal(CollectInstancesPayload{Region: region})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskCollectInstancesForRegion, payload), nil
}

// HandleCollectInstancesForRegionTask collects EC2 Instances for a specific Region.
func HandleCollectInstancesForRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectInstancesPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return collectInstancesForRegion(ctx, p.Region)
}

func collectInstancesForRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS instances ", "region", region)
	paginator := ec2.NewDescribeInstancesPaginator(
		awsclient.EC2,
		&ec2.DescribeInstancesInput{},
		func(params *ec2.DescribeInstancesPaginatorOptions) {
			params.Limit = int32(constants.PageSize)
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.Instance, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *ec2.Options) {
				o.Region = region
			},
		)
		if err != nil {
			slog.Error("could not describe instances", "region", region, "reason", err)
			return err
		}
		for _, reservation := range page.Reservations {
			items = append(items, reservation.Instances...)
		}
	}

	// Parse reservations and add to instances
	instances := make([]models.Instance, 0, len(items))
	for _, instance := range items {
		name := utils.FetchTag(instance.Tags, "Name")
		modelInstance := models.Instance{
			Name:         name,
			Arch:         string(instance.Architecture),
			InstanceID:   strings.StringFromPointer(instance.InstanceId),
			InstanceType: string(instance.InstanceType),
			State:        string(instance.State.Name),
			SubnetID:     strings.StringFromPointer(instance.SubnetId),
			VpcID:        strings.StringFromPointer(instance.VpcId),
			Platform:     strings.StringFromPointer(instance.PlatformDetails),
			RegionName:   region,
			ImageID:      strings.StringFromPointer(instance.ImageId),
		}
		instances = append(instances, modelInstance)
	}

	if len(instances) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&instances).
		On("CONFLICT (instance_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("arch = EXCLUDED.arch").
		Set("instance_type = EXCLUDED.instance_type").
		Set("state = EXCLUDED.state").
		Set("subnet_id = EXCLUDED.subnet_id").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("platform = EXCLUDED.platform").
		Set("region_name = EXCLUDED.region_name").
		Set("image_id = EXCLUDED.image_id").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert instances into db", "region", region, "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws instances", "region", region, "count", count)

	return nil
}

// HandleCollectInstancesTask collects the EC2 Instances from all known regions.
func HandleCollectInstancesTask(ctx context.Context, t *asynq.Task) error {
	return collectInstances(ctx)
}

func collectInstances(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := db.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "reason", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		instanceTask, err := NewCollectInstancesForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := asynqclient.Client.Enqueue(instanceTask)
		if err != nil {
			slog.Error(
				"could not enqueue task",
				"type", instanceTask.Type(),
				"region", r.Name,
				"reason", err,
			)
			continue
		}

		slog.Info(
			"enqueued task",
			"type", instanceTask.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
		)
	}

	return nil
}
