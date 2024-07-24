// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	awsclient "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/utils/strings"
)

// ErrMissingRegion is returned when an expected region name is missing.
var ErrMissingRegion = errors.New("missing region name")

const (
	// Asynq task type for collecting AWS regions
	TaskCollectRegions = "aws:task:collect-regions"
)

// NewAwsCollectRegionsTask creates a new task for collecting AWS regions.
func NewCollectRegionsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectRegions, nil)
}

// HandleAwsCollectRegionsTask is a handler function that collects AWS regions.
func HandleAwsCollectRegionsTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Collecting AWS regions")

	regionsOutput, err := awsclient.EC2.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		slog.Error("could not describe regions", "reason", err)
		return err
	}

	regions := make([]models.Region, 0, len(regionsOutput.Regions))
	for _, region := range regionsOutput.Regions {
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
	_, err = db.DB.NewInsert().
		Model(&regions).
		On("CONFLICT (name) DO UPDATE").
		Set("endpoint = EXCLUDED.endpoint").
		Set("opt_in_status = EXCLUDED.opt_in_status").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert regions into db", "reason", err)
		return err
	}

	return nil
}
