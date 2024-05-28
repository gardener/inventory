package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gardener/inventory/pkg/utils/strings"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/smithy-go/ptr"
	"github.com/gardener/inventory/pkg/aws/clients"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/hibiken/asynq"
)

const (
	// Asynq task type for collecting AWS regions
	AWS_COLLECT_AZS_REGION_TYPE = "aws:collect-azs-region"
	AWS_COLLECT_AZS_TYPE        = "aws:collect-azs"
)

type CollectAzsPayload struct {
	Region string `json:"region"`
}

// TODO: Shall it be triggered by the collect regions task?
func NewCollectAzsRegionTask(region string) *asynq.Task {
	if region == "" {
		slog.Error("region is required and cannot be empty")
		return nil
	}
	payload, err := json.Marshal(CollectAzsPayload{Region: region})
	if err != nil {
		slog.Error("could not marshal payload", "err", err)
		return nil
	}
	return asynq.NewTask(AWS_COLLECT_AZS_REGION_TYPE, payload)
}

func HandleCollectAzsRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectAzsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	return collectAzsRegion(ctx, p.Region)
}

// collect AWS availability zones.
func collectAzsRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS availability zones", "region", region)

	azsOutput, err := clients.Ec2.DescribeAvailabilityZones(ctx,
		&ec2.DescribeAvailabilityZonesInput{
			AllAvailabilityZones: ptr.Bool(false),
		},
		func(o *ec2.Options) {
			o.Region = region
		},
	)

	if err != nil {
		slog.Error("could not describe availability zones", "err", err)
		return err
	}

	azs := make([]models.AvailabilityZone, 0, len(azsOutput.AvailabilityZones))
	for _, az := range azsOutput.AvailabilityZones {
		slog.Info("Availability Zone", "name", *az.ZoneName, "region", *az.RegionName)
		modelAz := models.AvailabilityZone{
			ZoneID:             strings.StringFromPointer(az.ZoneId),
			Name:               strings.StringFromPointer(az.ZoneName),
			OptInStatus:        string(az.OptInStatus),
			State:              string(az.State),
			RegionName:         strings.StringFromPointer(az.RegionName),
			GroupName:          strings.StringFromPointer(az.GroupName),
			NetworkBorderGroup: strings.StringFromPointer(az.NetworkBorderGroup),
		}
		azs = append(azs, modelAz)

	}

	if len(azs) == 0 {
		return nil
	}
	_, err = clients.Db.NewInsert().
		Model(&azs).
		On("CONFLICT (zone_id) DO UPDATE").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert availability zones into db", "err", err)
		return err
	}

	return nil
}

// NewCollectAzsTask creates a new task for collecting AWS availability zones without specifying a region.
// It fetches the reqions from the database and triggers an aws:collect-azs-region task for each region.
func NewCollectAzsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_AZS_TYPE, nil)
}

func HandleCollectAzsTask(ctx context.Context, t *asynq.Task) error {
	return collectAzs(ctx)
}

func collectAzs(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := clients.Db.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "err", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		azsTask := NewCollectAzsRegionTask(r.Name)
		info, err := clients.Client.Enqueue(azsTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", azsTask.Type(), "err", err)
			continue
		}
		slog.Info("enqueued task", "type", azsTask.Type(), "id", info.ID, "queue", info.Queue)
	}
	return nil
}
