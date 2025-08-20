// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

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
	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectHostedZonesTask creates a new [asynq.Task] for collecting AWS
// Route 53 hosted zones, without specifying a payload.
func NewCollectHostedZonesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectHostedZones, nil)
}

// HandleCollectHostedZonesTask handles the task for collecting AWS
// Route 53 hosted zones
func HandleCollectHostedZonesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting hosted zones for all known accounts.
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

	return collectHostedZones(ctx, payload)
}

// enqueueCollectHostedZones enqueues tasks for collecting AWS route 53 hosted zones for the known
// accounts.
func enqueueCollectHostedZones(ctx context.Context) error {
	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	err := awsclients.Route53Clientset.Range(func(accountID string, _ *awsclients.Client[*route53.Client]) error {
		payload := CollectHostedZonesPayload{
			AccountID: accountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS hosted zones",
				"account_id", accountID,
				"reason", err,
			)

			return err
		}

		task := asynq.NewTask(TaskCollectHostedZones, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"account_id", accountID,
				"reason", err,
			)

			return err
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"account_id", accountID,
		)

		return nil
	})


	return err
}

// collectHostedZones collects the AWS route 53 hosted zones from the specified
// account ID using the associated client
func collectHostedZones(ctx context.Context, payload CollectHostedZonesPayload) error {
	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
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
	)
	key := metrics.Key(TaskCollectHostedZones, payload.AccountID)
	metrics.DefaultCollector.AddMetric(key, metric)

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS hosted zones",
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
			},
		)

		if err != nil {
			logger.Error(
				"could not describe hosted zones",
				"account_id", payload.AccountID,
				"reason", err,
			)

			return err
		}
		items = append(items, page.HostedZones...)
	}

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

		hostedZoneID := awsutils.CutHostedZonePrefix(ptr.StringFromPointer(item.Id))

		hostedZone := models.HostedZone{
			AccountID:              payload.AccountID,
			HostedZoneID:           hostedZoneID,
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
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}
