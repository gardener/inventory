// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/aws/constants"
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
	// TaskCollectHostedZones is the name of the task for collecting
	// AWS Route 53 hosted zones.
	TaskCollectHostedZones = "aws:task:collect-hosted-zones"
)

// CollectHostedZonesPayload represents the payload for collecting AWS
// Route 53 hosted zones
type CollectHostedZonesPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectHostedZoneTask creates a new [asynq.Task] for collecting AWS
// Route 53 hosted zones, without specifying a payload.
func NewCollectHostedZonesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectHostedZones, nil)
}

// HandleCollectHostedZonesTask handles the task for collecting AWS
// Route 53 hosted zones
func HandleCollectHostedZonesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting hosted zones from all known regions and their respective accounts.
	data := t.Payload()
	if data == nil {
		return enqueueCollectHostedZones(ctx)
	}

	var payload CollectHostedZonesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	return collectHostedZones(ctx, payload)
}

// enqueueCollectHostedZones enqueues tasks for collecting AWS route 53 hosted zones for the known
// regions and accounts.
func enqueueCollectHostedZones(ctx context.Context) error {
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Enqueue hosted zone collection for each region
	for _, r := range regions {
		if !awsclients.Route53Clientset.Exists(r.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", r.Name,
				"account_id", r.AccountID,
			)

			continue
		}

		payload := CollectHostedZonesPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS hosted zones",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectHostedZones, data)
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

// collectHostedZones collects the AWS route 53 hosted zones from the specified region using the client
// associated with the given AccountID from the payload.
func collectHostedZones(ctx context.Context, payload CollectHostedZonesPayload) error {
	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	client, ok := awsclients.Route53Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	var count int64
	metric := prometheus.MustNewConstMetric(
		hostedZonesDesc,
		prometheus.GaugeValue,
		float64(count),
		payload.AccountID,
		payload.Region,
	)
	key := metrics.Key(TaskCollectHostedZones, payload.AccountID, payload.Region)
	metrics.DefaultCollector.AddMetric(key, metric)

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS hosted zones",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	paginator := route53.NewListHostedZonesPaginator(
		client.Client,
		&route53.ListHostedZonesInput{},
		func(opts *route53.ListHostedZonesPaginatorOptions) {
			opts.Limit = int32(constants.PageSize)
			opts.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.HostedZone, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *route53.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not describe hosted zones",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return err
		}
		items = append(items, page.HostedZones...)
	}

	// Create model instances from the collected data
	hostedZones := make([]models.HostedZone, 0, len(items))
	for _, item := range items {
		var description string
		if item.LinkedService != nil {
			description = ptr.StringFromPointer(item.LinkedService.Description)
		}

		privateZone := false
		comment := ""

		if item.Config != nil {
			privateZone = item.Config.PrivateZone
			comment = ptr.StringFromPointer(item.Config.Comment)
		}

		hostedZone := models.HostedZone{
			RegionName:             payload.Region,
			AccountID:              payload.AccountID,
			HostedZoneID:           ptr.StringFromPointer(item.Id),
			Name:                   ptr.StringFromPointer(item.Name),
			Description:            description,
			CallerReference:        ptr.StringFromPointer(item.CallerReference),
			Comment:                comment,
			IsPrivate:              privateZone,
			ResourceRecordSetCount: *item.ResourceRecordSetCount,
		}

		hostedZones = append(hostedZones, hostedZone)
	}

	if len(hostedZones) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&hostedZones).
		On("CONFLICT (hosted_zone_id, account_id) DO UPDATE").
		Set("region_name = EXCLUDED.region_name").
		Set("name = EXCLUDED.name").
		Set("description = EXCLUDED.description").
		Set("caller_reference = EXCLUDED.caller_reference").
		Set("comment = EXCLUDED.comment").
		Set("is_private = EXCLUDED.is_private").
		Set("resource_record_set_count = EXCLUDED.resource_record_set_count").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert hosted zones into db",
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
		"populated aws hosted zones",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}
