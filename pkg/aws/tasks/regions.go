package tasks

import (
	"context"
	"log/slog"

	"github.com/gardener/inventory/pkg/aws/clients"
	"github.com/gardener/inventory/pkg/aws/models"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"
)

const (
	// Asynq task type for collecting AWS regions
	AWS_COLLECT_REGIONS_TYPE = "aws:collect-regions"
)

// NewAwsCollectRegionsTask creates a new task for collecting AWS regions.
func NewAwsCollectRegionsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_REGIONS_TYPE, nil)
}

// HandleAwsCollectRegionsTask is a handler function that collects AWS regions.
func HandleAwsCollectRegionsTask(ctx context.Context, t *asynq.Task) error {
	err := collectRegions(ctx)
	if err != nil {
		return err
	}
	return nil
}

func collectRegions(ctx context.Context) error {
	slog.Info("Collecting AWS regions")

	regionsOutput, err := clients.Ec2.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})

	if err != nil {
		slog.Error("could not describe regions", "err", err)
		return err
	}

	regions := make([]models.Region, 0, len(regionsOutput.Regions))
	for _, region := range regionsOutput.Regions {
		slog.Info("Region", "name", *region.RegionName)
		modelRegion := models.Region{
			Name:        *region.RegionName,
			Endpoint:    *region.Endpoint,
			OptInStatus: *region.OptInStatus,
		}
		regions = append(regions, modelRegion)

		// Create asynq task for collecting availability zones
		azsTask := NewAwsCollectAzsTask(*region.RegionName)
		info, err := clients.Client.Enqueue(azsTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", azsTask.Type(), "err", err)
			continue
		}
		slog.Info("enqueued task", "type", azsTask.Type(), "id", info.ID, "queue", info.Queue)

	}

	//Bulk insert regions into db
	_, err = clients.Db.NewInsert().
		Model(&regions).
		On("CONFLICT (id) DO UPDATE").
		Ignore().
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert regions into db", "err", err)
		return err
	}

	return nil
}
