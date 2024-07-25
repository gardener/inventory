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
	"github.com/aws/smithy-go/ptr"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclient "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	// Asynq task type for collecting AWS regions
	TaskCollectAvailabilityZonesForRegion = "aws:task:collect-azs-region"
	TaskCollectAvailabilityZones          = "aws:task:collect-azs"
)

type CollectAzsPayload struct {
	Region string `json:"region"`
}

// NewCollectAzsForRegionTask creates a new task for collecting AWS Availability
// Zones for a given region.
func NewCollectAzsForRegionTask(region string) (*asynq.Task, error) {
	if region == "" {
		return nil, ErrMissingRegion
	}

	payload, err := json.Marshal(CollectAzsPayload{Region: region})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TaskCollectAvailabilityZonesForRegion, payload), nil
}

// HandleCollectAzsForRegionTask is the task handler which collects Availability
// Zones for a given region.
func HandleCollectAzsForRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectAzsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	return collectAzsForRegion(ctx, p.Region)
}

// Collect AWS availability zones for a given region.
func collectAzsForRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS availability zones", "region", region)

	azsOutput, err := awsclient.EC2.DescribeAvailabilityZones(ctx,
		&ec2.DescribeAvailabilityZonesInput{
			AllAvailabilityZones: ptr.Bool(false),
		},
		func(o *ec2.Options) {
			o.Region = region
		},
	)

	if err != nil {
		slog.Error("could not describe availability zones", "reason", err)
		return err
	}

	azs := make([]models.AvailabilityZone, 0, len(azsOutput.AvailabilityZones))
	for _, az := range azsOutput.AvailabilityZones {
		modelAz := models.AvailabilityZone{
			ZoneID:             strings.StringFromPointer(az.ZoneId),
			ZoneType:           strings.StringFromPointer(az.ZoneType),
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
	_, err = db.DB.NewInsert().
		Model(&azs).
		On("CONFLICT (zone_id) DO UPDATE").
		Set("zone_type = EXCLUDED.zone_type").
		Set("name = EXCLUDED.name").
		Set("opt_in_status = EXCLUDED.opt_in_status").
		Set("state = EXCLUDED.state").
		Set("region_name = EXCLUDED.region_name").
		Set("group_name = EXCLUDED.group_name").
		Set("network_border_group = EXCLUDED.network_border_group").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)
	if err != nil {
		slog.Error("could not insert availability zones into db", "region", region, "reason", err)
		return err
	}

	return nil
}

// NewCollectAzsTask creates a new task for collecting AWS availability zones without specifying a region.
// It fetches the reqions from the database and triggers an aws:collect-azs-region task for each region.
func NewCollectAzsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectAvailabilityZones, nil)
}

// HandleCollectAzsTask handles the task for collecting all AZs for all Regions.
func HandleCollectAzsTask(ctx context.Context, t *asynq.Task) error {
	return collectAzs(ctx)
}

func collectAzs(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := db.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "reason", err)
		return err
	}

	for _, r := range regions {
		// Trigger Asynq task for each region
		azsTask, err := NewCollectAzsForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := asynqclient.Client.Enqueue(azsTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", azsTask.Type(), "reason", err)
			continue
		}

		slog.Info("enqueued task", "type", azsTask.Type(), "id", info.ID, "queue", info.Queue)
	}

	return nil
}
