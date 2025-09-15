// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
)

const (
	// TaskCollectDHCPOptionSets is the name of the task for collecting AWS DHCP option sets.
	TaskCollectDHCPOptionSets = "aws:task:collect-dhcp-option-sets"
)

// CollectDHCPOptionSetsPayload is the payload, which is used for collecting AWS DHCP option sets.
type CollectDHCPOptionSetsPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectDHCPOptionSetsTask creates a new [asynq.Task] for collecting AWS DHCP option sets without
// specifying a payload.
func NewCollectDHCPOptionSetsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectDHCPOptionSets, nil)
}

// HandleCollectDHCPOptionSetsTask handles the task for collecting AWS DHCP option sets.
func HandleCollectDHCPOptionSetsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting DHCP option sets for all known regions.
	data := t.Payload()
	if data == nil {
		return enqueueCollectDHCPOptionSets(ctx)
	}

	var payload CollectDHCPOptionSetsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	return collectDHCPOptionSets(ctx, payload)
}

// enqueueCollectDHCPOptionSets enqueues tasks for collecting AWS DHCP option sets from all known
// regions by creating payload with the respective region and account id.
func enqueueCollectDHCPOptionSets(ctx context.Context) error {
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Enqueue task for each region
	for _, r := range regions {
		if !awsclients.EC2Clientset.Exists(r.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", r.Name,
				"account_id", r.AccountID,
			)

			continue
		}

		payload := CollectDHCPOptionSetsPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS DHCP option set",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectDHCPOptionSets, data)
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

// collectDHCPOptionSets collects the AWS DHCP option sets from the specified payload region using the
// client associated with the specified AccountID.
func collectDHCPOptionSets(ctx context.Context, payload CollectDHCPOptionSetsPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			dhcpOptionSetDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.AccountID,
			payload.Region,
		)
		key := metrics.Key(TaskCollectDHCPOptionSets, payload.AccountID, payload.Region)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS DHCP option sets",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	paginator := ec2.NewDescribeDhcpOptionsPaginator(
		client.Client,
		&ec2.DescribeDhcpOptionsInput{},
		func(params *ec2.DescribeDhcpOptionsPaginatorOptions) {
			params.Limit = int32(constants.PageSize)
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.DhcpOptions, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *ec2.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not describe DHCP option sets",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return err
		}
		items = append(items, page.DhcpOptions...)
	}

	dhcpOptionSets := make([]models.DHCPOptionSet, 0, len(items))
	for _, set := range items {
		name := awsutils.FetchTag(set.Tags, "Name")

		if set.DhcpOptionsId == nil {
			logger.Warn(
				"empty DHCP option set id",
				"name", name,
			)

			continue
		}

		item := models.DHCPOptionSet{
			Name:       name,
			AccountID:  payload.AccountID,
			SetID:      *set.DhcpOptionsId,
			RegionName: payload.Region,
		}
		dhcpOptionSets = append(dhcpOptionSets, item)
	}

	if len(dhcpOptionSets) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&dhcpOptionSets).
		On("CONFLICT (set_id, account_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert DHCP option sets into db",
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
		"populated AWS DHCP option sets",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}
