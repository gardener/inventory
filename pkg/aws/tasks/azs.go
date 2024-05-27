package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/smithy-go/ptr"
	"github.com/gardener/inventory/pkg/aws/clients"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/hibiken/asynq"
)

const (
	// Asynq task type for collecting AWS regions
	AWS_COLLECT_AZS_TYPE = "aws:collect-azs"
)

type CollectAzsPayload struct {
	Region string `json:"region"`
}

// TODO: Shall it be triggered by the collect regions task?
func NewAwsCollectAzsTask(region string) *asynq.Task {
	if region == "" {
		slog.Error("region is required and cannot be empty")
		return nil
	}
	payload, err := json.Marshal(CollectAzsPayload{Region: region})
	if err != nil {
		slog.Error("could not marshal payload", "err", err)
		return nil
	}
	return asynq.NewTask(AWS_COLLECT_AZS_TYPE, payload)
}

func HandleAwsCollectAzsTask(ctx context.Context, t *asynq.Task) error {
	var p CollectAzsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	return collectAzs(ctx, p.Region)
}

// collect AWS availability zones.
func collectAzs(ctx context.Context, region string) error {
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
			ZoneID:             *az.ZoneId,
			Name:               *az.ZoneName,
			OptInStatus:        string(az.OptInStatus),
			State:              string(az.State),
			RegionName:         *az.RegionName,
			GroupName:          *az.GroupName,
			NetworkBorderGroup: *az.NetworkBorderGroup,
		}
		azs = append(azs, modelAz)

	}
	_, err = clients.Db.NewInsert().
		Model(&azs).
		On("CONFLICT (id) DO UPDATE").
		Ignore().
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert availability zones into db", "err", err)
		return err
	}

	return nil
}
