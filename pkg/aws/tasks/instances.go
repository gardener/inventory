package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	awsclients "github.com/gardener/inventory/pkg/aws/clients"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/aws/utils"
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_INSTANCES_TYPE        = "aws:task:collect-instances"
	AWS_COLLECT_INSTANCES_REGION_TYPE = "aws:task:collect-instances-region"
)

type CollectInstancesPayload struct {
	Region string `json:"region"`
}

// NewCollectInstancesTask creates a new task for collecting EC2 Instances from
// all AWS Regions.
func NewCollectInstancesTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_INSTANCES_TYPE, nil)
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

	return asynq.NewTask(AWS_COLLECT_INSTANCES_REGION_TYPE, payload), nil
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

	instancesOutput, err := awsclients.Ec2.DescribeInstances(ctx,
		&ec2.DescribeInstancesInput{},
		func(o *ec2.Options) {
			o.Region = region
		},
	)

	if err != nil {
		slog.Error("could not describe instances", "err", err)
		return err
	}

	count := 0
	for _, reservation := range instancesOutput.Reservations {
		count = count + len(reservation.Instances)
	}
	slog.Info("found instances", "count", count, "region", region)

	// Parse reservations and add to instances
	instances := make([]models.Instance, 0, count)

	for _, reservation := range instancesOutput.Reservations {
		for _, instance := range reservation.Instances {
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
			}
			instances = append(instances, modelInstance)
		}
	}

	if len(instances) == 0 {
		return nil
	}

	_, err = clients.Db.NewInsert().
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
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert instances into db", "err", err)
		return err
	}

	return nil
}

// HandleCollectInstancesTask collects the EC2 Instances from all known regions.
func HandleCollectInstancesTask(ctx context.Context, t *asynq.Task) error {
	return collectInstances(ctx)
}

func collectInstances(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := clients.Db.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "err", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		instanceTask, err := NewCollectInstancesForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := clients.Client.Enqueue(instanceTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", instanceTask.Type(), "err", err)
			continue
		}

		slog.Info("enqueued task", "type", instanceTask.Type(), "id", info.ID, "queue", info.Queue)
	}

	return nil
}
