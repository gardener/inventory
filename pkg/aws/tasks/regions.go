package tasks

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	awsclients "github.com/gardener/inventory/pkg/aws/clients"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// Asynq task type for collecting AWS regions
	AWS_COLLECT_REGIONS_TYPE = "aws:task:collect-regions"
)

// NewAwsCollectRegionsTask creates a new task for collecting AWS regions.
func NewAwsCollectRegionsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_REGIONS_TYPE, nil)
}

// HandleAwsCollectRegionsTask is a handler function that collects AWS regions.
func HandleAwsCollectRegionsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting AWS regions")

	regionsOutput, err := awsclients.Ec2.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		slog.Error("could not describe regions", "err", err)
		return err
	}

	regions := make([]models.Region, 0, len(regionsOutput.Regions))
	for _, region := range regionsOutput.Regions {
		slog.Info("Region", "name", strings.StringFromPointer(region.RegionName))
		modelRegion := models.Region{
			Name:        strings.StringFromPointer(region.RegionName),
			Endpoint:    strings.StringFromPointer(region.Endpoint),
			OptInStatus: strings.StringFromPointer(region.OptInStatus),
		}
		regions = append(regions, modelRegion)
	}

	if len(regions) == 0 {
		return nil
	}

	// Bulk insert regions into db
	_, err = clients.Db.NewInsert().
		Model(&regions).
		On("CONFLICT (name) DO UPDATE").
		Set("endpoint = EXCLUDED.endpoint").
		Set("opt_in_status = EXCLUDED.opt_in_status").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert regions into db", "err", err)
		return err
	}

	return nil
}
