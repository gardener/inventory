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
	"github.com/gardener/inventory/pkg/aws/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclients "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	dbutils "github.com/gardener/inventory/pkg/utils/db"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectRecords is the name of the task for collecting
	// AWS Route 53 DNS records from hosted zones.
	TaskCollectRecords = "aws:task:collect-record"
)

// CollectRecordsPayload represents the payload for collecting AWS
// Route 53 DNS records from a specific hosted zone
type CollectRecordsPayload struct {
	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`

	// HostedZoneID specifies the hosted zone from which to collect records.
	HostedZoneID string `json:"hosted_zone_id" yaml:"hosted_zone_id"`
}

// NewCollectRecordsTask creates a new [asynq.Task] for collecting AWS
// Route 53 DNS records, without specifying a payload.
func NewCollectRecordsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectRecords, nil)
}

// HandleCollectRecordsTask handles the task for collecting AWS
// Route 53 DNS records
func HandleCollectRecordsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting records from all known hosted zones.
	data := t.Payload()
	if data == nil {
		return enqueueCollectRecords(ctx)
	}

	var payload CollectRecordsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.HostedZoneID == "" {
		return asynqutils.SkipRetry(fmt.Errorf("hosted zone ID is required"))
	}

	return collectRecords(ctx, payload)
}

// enqueueCollectRecords enqueues tasks for collecting
// AWS Route53 DNS records for all known hosted zones.
func enqueueCollectRecords(ctx context.Context) error {
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
				"account_id", hz.AccountID,
				"hosted_zone_id", hz.HostedZoneID,
			)

			continue
		}

		payload := CollectRecordsPayload{
			AccountID:    hz.AccountID,
			HostedZoneID: hz.HostedZoneID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS dns records",
				"account_id", hz.AccountID,
				"hosted_zone_id", hz.HostedZoneID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectRecords, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
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
			"account_id", hz.AccountID,
			"hosted_zone_id", hz.HostedZoneID,
		)
	}

	return nil
}

// collectRecords collects the AWS Route53 DNS records from the specified hosted zone
// using the client associated with the given AccountID from the payload.
func collectRecords(ctx context.Context, payload CollectRecordsPayload) error {
	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
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
			recordsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.AccountID,
			payload.HostedZoneID,
		)
		key := metrics.Key(TaskCollectRecords, payload.AccountID, payload.HostedZoneID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS Route53 dns records",
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

	recordSets := make([]types.ResourceRecordSet, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
		)
		if err != nil {
			logger.Error(
				"could not list AWS dns records",
				"account_id", payload.AccountID,
				"hosted_zone_id", payload.HostedZoneID,
				"reason", err,
			)

			return err
		}

		recordSets = append(recordSets, page.ResourceRecordSets...)
	}

	records := make([]models.ResourceRecord, 0)
	for _, set := range recordSets {
		var dnsName string
		var isAlias, evaluateHealth bool
		if set.AliasTarget != nil && set.ResourceRecords != nil && len(set.ResourceRecords) > 0 {
			logger.Warn(`ambiguous state found in aws dns:
				record is both alias and has a records set.
				Treating as alias.`,
				"account_id", payload.AccountID,
				"hosted_zone_id", payload.HostedZoneID,
				"fqdn", set.Name,
			)
		}

		name := utils.RestoreAsteriskPrefix(ptr.StringFromPointer(set.Name))
		if set.AliasTarget != nil {
			isAlias = true
			dnsName = ptr.StringFromPointer(set.AliasTarget.DNSName)
			evaluateHealth = set.AliasTarget.EvaluateTargetHealth
			record := models.ResourceRecord{
				AccountID:      payload.AccountID,
				HostedZoneID:   payload.HostedZoneID,
				Name:           name,
				IsAlias:        isAlias,
				Type:           string(set.Type),
				TTL:            set.TTL,
				SetIdentifier:  ptr.StringFromPointer(set.SetIdentifier),
				EvaluateHealth: evaluateHealth,
				Value:          dnsName,
			}

			records = append(records, record)
		} else {
			for _, rr := range set.ResourceRecords {
				record := models.ResourceRecord{
					AccountID:      payload.AccountID,
					HostedZoneID:   payload.HostedZoneID,
					Name:           name,
					IsAlias:        isAlias,
					Type:           string(set.Type),
					TTL:            set.TTL,
					SetIdentifier:  ptr.StringFromPointer(set.SetIdentifier),
					EvaluateHealth: evaluateHealth,
					Value:          ptr.StringFromPointer(rr.Value),
				}
				records = append(records, record)
			}
		}
	}

	out, err := db.DB.NewInsert().
		Model(&records).
		On("CONFLICT (account_id, hosted_zone_id, name, type, set_identifier, value) DO UPDATE").
		Set("is_alias = EXCLUDED.is_alias").
		Set("ttl = EXCLUDED.ttl").
		Set("evaluate_health = EXCLUDED.evaluate_health").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert dns records into db",
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
		"populated AWS Route53 dns records",
		"account_id", payload.AccountID,
		"hosted_zone_id", payload.HostedZoneID,
		"slice count", len(records),
		"count", count,
	)

	return nil
}
