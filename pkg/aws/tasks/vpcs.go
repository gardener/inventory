// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
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
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectVPCs is the name of the task for collecting AWS VPCs.
	TaskCollectVPCs = "aws:task:collect-vpcs"
)

// CollectVPCsPayload is the payload, which is used for collecting AWS VPCs.
type CollectVPCsPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectVPCsTask creates a new [asynq.Task] for collecting AWS VPCs without
// specifying a payload.
func NewCollectVPCsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectVPCs, nil)
}

// HandleCollectVPCsTask handles the task for collecting AWS VPCs.
func HandleCollectVPCsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting VPCs for all known regions.
	data := t.Payload()
	if data == nil {
		return enqueueCollectVPCs(ctx)
	}

	var payload CollectVPCsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	return collectVPCs(ctx, payload)
}

// enqueueCollectVPCs enqueues tasks for collecting AWS VPCs from all known
// regions by creating payload with the respective region and account id.
func enqueueCollectVPCs(ctx context.Context) error {
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

		payload := CollectVPCsPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS VPC",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectVPCs, data)
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

// collectVPCs collects the AWS VPCs from the specified payload region using the
// client associated with the specified AccountID.
func collectVPCs(ctx context.Context, payload CollectVPCsPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			vpcsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.AccountID,
			payload.Region,
		)
		key := metrics.Key(TaskCollectVPCs, payload.AccountID, payload.Region)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS VPCs",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	paginator := ec2.NewDescribeVpcsPaginator(
		client.Client,
		&ec2.DescribeVpcsInput{},
		func(params *ec2.DescribeVpcsPaginatorOptions) {
			params.Limit = int32(constants.PageSize)
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.Vpc, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *ec2.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not describe VPCs",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return err
		}
		items = append(items, page.Vpcs...)
	}

	vpcs := make([]models.VPC, 0, len(items))
	for _, vpc := range items {
		name := awsutils.FetchTag(vpc.Tags, "Name")
		item := models.VPC{
			Name:            name,
			AccountID:       payload.AccountID,
			VpcID:           ptr.StringFromPointer(vpc.VpcId),
			State:           string(vpc.State),
			IPv4CIDR:        ptr.StringFromPointer(vpc.CidrBlock),
			IPv6CIDR:        "", // TODO: fetch IPv6 CIDR
			IsDefault:       ptr.Value(vpc.IsDefault, false),
			OwnerID:         ptr.StringFromPointer(vpc.OwnerId),
			DHCPOptionSetID: ptr.StringFromPointer(vpc.DhcpOptionsId),
			RegionName:      payload.Region,
		}
		vpcs = append(vpcs, item)
	}

	if len(vpcs) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&vpcs).
		On("CONFLICT (vpc_id, account_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("state = EXCLUDED.state").
		Set("ipv4_cidr = EXCLUDED.ipv4_cidr").
		Set("ipv6_cidr = EXCLUDED.ipv6_cidr").
		Set("is_default = EXCLUDED.is_default").
		Set("owner_id = EXCLUDED.owner_id").
		Set("dhcp_option_set_id = EXCLUDED.dhcp_option_set_id").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert VPCs into db",
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
		"populated aws vpcs",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}
