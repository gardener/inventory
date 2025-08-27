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
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	dbutils "github.com/gardener/inventory/pkg/utils/db"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectRecordSets is the name of the task for collecting
	// AWS Route 53 DNS records from hosted zones.
	TaskCollectRecordSets = "aws:task:collect-record-sets"
)

// CollectRecordSetsPayload represents the payload for collecting AWS
// Route 53 DNS records from a specific hosted zone
type CollectRecordSetsPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`

	// HostedZoneID specifies the hosted zone from which to collect records.
	HostedZoneID string `json:"hosted_zone_id" yaml:"hosted_zone_id"`
}

// NewCollectRecordSetsTask creates a new [asynq.Task] for collecting AWS
// Route 53 DNS records, without specifying a payload.
func NewCollectRecordSetsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectRecordSets, nil)
}

// HandleCollectRecordSetsTask handles the task for collecting AWS
// Route 53 DNS records
func HandleCollectRecordSetsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting records from all known hosted zones.
	data := t.Payload()
	if data == nil {
		return enqueueCollectRecordSets(ctx)
	}

	var payload CollectRecordSetsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	if payload.HostedZoneID == "" {
		return asynqutils.SkipRetry(fmt.Errorf("hosted zone ID is required"))
	}

	return collectRecordSets(ctx, payload)
}

// enqueueCollectRecordSets enqueues tasks for collecting
// AWS Route53 DNS records for all known hosted zones.
func enqueueCollectRecordSets(ctx context.Context) error {
	hostedZones, err := dbutils.GetResourcesFromDB[models.HostedZone](ctx)
	if err != nil {
		return fmt.Errorf("failed to get hosted zones: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	for _, hz := range hostedZones {
		if !awsclients.Route53Clientset.Exists(hz.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", hz.RegionName,
				"account_id", hz.AccountID,
				"hosted_zone_id", hz.HostedZoneID,
			)

			continue
		}

		payload := CollectRecordSetsPayload{
			Region:       hz.RegionName,
			AccountID:    hz.AccountID,
			HostedZoneID: hz.HostedZoneID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS record sets",
				"region", hz.RegionName,
				"account_id", hz.AccountID,
				"hosted_zone_id", hz.HostedZoneID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectRecordSets, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"region", hz.RegionName,
				"account_id", hz.AccountID,
				"hosted_zone_id", hz.HostedZoneID,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", hz.RegionName,
			"account_id", hz.AccountID,
			"hosted_zone_id", hz.HostedZoneID,
		)
	}

	return nil
}

// collectRecordSets collects the AWS Route53 DNS records from the specified hosted zone
// using the client associated with the given AccountID from the payload.
func collectRecordSets(ctx context.Context, payload CollectRecordSetsPayload) error {
	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	if payload.HostedZoneID == "" {
		return asynqutils.SkipRetry(fmt.Errorf("hosted zone ID is required"))
	}

	client, ok := awsclients.Route53Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			recordSetsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.AccountID,
			payload.Region,
			payload.HostedZoneID,
		)
		key := metrics.Key(TaskCollectRecordSets, payload.AccountID, payload.Region, payload.HostedZoneID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS Route53 record sets",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"hosted_zone_id", payload.HostedZoneID,
	)

	paginator := route53.NewListResourceRecordSetsPaginator(
		client.Client,
		&route53.ListResourceRecordSetsInput{
			HostedZoneId: &payload.HostedZoneID,
		},
		func(opts *route53.ListResourceRecordSetsPaginatorOptions) {
			opts.Limit = int32(constants.PageSize)
			opts.StopOnDuplicateToken = true
		},
	)

	items := make([]types.ResourceRecordSet, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *route53.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not list resource record sets",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"hosted_zone_id", payload.HostedZoneID,
				"reason", err,
			)

			return err
		}
		items = append(items, page.ResourceRecordSets...)
	}

	recordSets := make([]models.RecordSet, 0, len(items))
	for _, item := range items {
		var dnsName/*, aliasHostedZoneID*/ string
		var isAlias, evaluateHealth bool
		if item.AliasTarget != nil {
			isAlias = true
			dnsName = ptr.StringFromPointer(item.AliasTarget.DNSName)
			evaluateHealth = item.AliasTarget.EvaluateTargetHealth
			// aliasHostedZoneID = ptr.StringFromPointer(item.AliasTarget.HostedZoneId)
		}

		recordSet := models.RecordSet{
			RegionName:     payload.Region,
			AccountID:      payload.AccountID,
			HostedZoneID:   payload.HostedZoneID,
			Name:           ptr.StringFromPointer(item.Name),
			IsAlias:        isAlias,
			Type:           string(item.Type),
			TTL:            item.TTL,
			SetIdentifier:  ptr.StringFromPointer(item.SetIdentifier),
			AliasDNSName:   dnsName,
			EvaluateHealth: evaluateHealth,
		}

		recordSets = append(recordSets, recordSet)
	}

	out, err := db.DB.NewInsert().
		Model(&recordSets).
		On("CONFLICT (account_id, hosted_zone_id, name, type, set_identifier) DO UPDATE").
		Set("region_name = EXCLUDED.region_name").
		Set("is_alias = EXCLUDED.is_alias").
		Set("ttl = EXCLUDED.ttl").
		Set("alias_dns_name = EXCLUDED.alias_dns_name").
		Set("evaluate_health = EXCLUDED.evaluate_health").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert record sets into db",
			"region", payload.Region,
			"account_id", payload.AccountID,
			"hosted_zone_id", payload.HostedZoneID,
			"reason", err,
		)

		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated AWS Route53 record sets",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"hosted_zone_id", payload.HostedZoneID,
		"count", count,
	)

	return nil
}
