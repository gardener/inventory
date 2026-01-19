// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/aws/models"
	awsutils "github.com/gardener/inventory/pkg/aws/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectAvailabilityZones is the name of the task for collecting
	// AWS AZs.
	TaskCollectAvailabilityZones = "aws:task:collect-azs"
)

// CollectAvailabilityZonesPayload is the payload, which is used for collecting
// AWS AZs.
type CollectAvailabilityZonesPayload struct {
	// Region is the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectAvailabilityZonesTask creates a new [asynq.Task] for collecting AWS
// Availability Zones without specifying a payload.
func NewCollectAvailabilityZonesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectAvailabilityZones, nil)
}

// HandleCollectAvailabilityZonesTask handles the task for collecting AWS AZs.
func HandleCollectAvailabilityZonesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting the Availability Zones for all known regions.
	data := t.Payload()
	if data == nil {
		return enqueueCollectAvailabilityZones(ctx)
	}

	// Collect the AZs from the specified region using the specified account
	var payload CollectAvailabilityZonesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	return collectAvailabilityZones(ctx, payload)
}

// collectAvailabilityZones collects the AWS AZs for the specified region in the
// payload, using the respective client associated with the given Account ID.
func collectAvailabilityZones(ctx context.Context, payload CollectAvailabilityZonesPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			zonesDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.AccountID,
			payload.Region,
		)
		key := metrics.Key(TaskCollectAvailabilityZones, payload.AccountID, payload.Region)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS availability zones",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	result, err := client.Client.DescribeAvailabilityZones(ctx,
		&ec2.DescribeAvailabilityZonesInput{
			AllAvailabilityZones: ptr.To(false),
		},
		func(o *ec2.Options) {
			o.Region = payload.Region
		},
	)

	if err != nil {
		logger.Error(
			"could not describe availability zones",
			"region", payload.Region,
			"account_id", payload.AccountID,
			"reason", err,
		)

		return awsutils.MaybeSkipRetry(err)
	}

	items := make([]models.AvailabilityZone, 0, len(result.AvailabilityZones))
	for _, item := range result.AvailabilityZones {
		item := models.AvailabilityZone{
			ZoneID:             ptr.StringFromPointer(item.ZoneId),
			AccountID:          payload.AccountID,
			ZoneType:           ptr.StringFromPointer(item.ZoneType),
			Name:               ptr.StringFromPointer(item.ZoneName),
			OptInStatus:        string(item.OptInStatus),
			State:              string(item.State),
			RegionName:         ptr.StringFromPointer(item.RegionName),
			GroupName:          ptr.StringFromPointer(item.GroupName),
			NetworkBorderGroup: ptr.StringFromPointer(item.NetworkBorderGroup),
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (zone_id, account_id) DO UPDATE").
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
		logger.Error(
			"could not insert availability zones into db",
			"region", payload.Region,
			"account_id", payload.AccountID,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated aws availability zones",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}

// enqueueCollectAvailabilityZones enqueues tasks for collecting AWS AZs for all
// known regions by specifying the respective payload for each region and
// account.
func enqueueCollectAvailabilityZones(ctx context.Context) error {
	// Get the known regions
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Enqueue a task for each region
	for _, r := range regions {
		if !awsclients.EC2Clientset.Exists(r.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", r.Name,
				"account_id", r.AccountID,
			)

			continue
		}

		payload := CollectAvailabilityZonesPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS availability zone",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectAvailabilityZones, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
			"account_id", r.AccountID,
		)
	}

	return nil
}
