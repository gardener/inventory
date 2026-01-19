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
	"github.com/gardener/inventory/pkg/utils"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

const (
	// TaskCollectSubnets is the name of the task for collecting AWS
	// Subnets.
	TaskCollectSubnets = "aws:task:collect-subnets"
)

// CollectSubnetsPayload is the payload, which is used to collect AWS subnets.
type CollectSubnetsPayload struct {
	// Region is the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectSubnetsTask creates a new [asynq.Task] for collecting AWS Subnets,
// without specifying a payload.
func NewCollectSubnetsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectSubnets, nil)
}

// HandleCollectSubnetsTask collects handles the task for collecting AWS
// Subnets.
func HandleCollectSubnetsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting the subnets for all known regions.
	data := t.Payload()
	if data == nil {
		return enqueueCollectSubnets(ctx)
	}

	var payload CollectSubnetsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	return collectSubnets(ctx, payload)
}

func enqueueCollectSubnets(ctx context.Context) error {
	// Get the known regions and enqueue a task for each
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)
	for _, r := range regions {
		if !awsclients.EC2Clientset.Exists(r.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", r.Name,
				"account_id", r.AccountID,
			)

			continue
		}

		payload := CollectSubnetsPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS subnets",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}
		task := asynq.NewTask(TaskCollectSubnets, data)
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

// collectSubnets collects the AWS Subnets for the specified region and using
// the client associated with the given account id from the payload.
func collectSubnets(ctx context.Context, payload CollectSubnetsPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS subnets",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	paginator := ec2.NewDescribeSubnetsPaginator(
		client.Client,
		&ec2.DescribeSubnetsInput{},
		func(params *ec2.DescribeSubnetsPaginatorOptions) {
			params.Limit = int32(constants.PageSize)
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.Subnet, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *ec2.Options) {
				o.Region = payload.Region
			},
		)
		if err != nil {
			logger.Error(
				"could not describe subnets",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return awsutils.MaybeSkipRetry(err)
		}
		items = append(items, page.Subnets...)
	}

	subnets := make([]models.Subnet, 0, len(items))
	for _, s := range items {
		name := awsutils.FetchTag(s.Tags, "Name")
		item := models.Subnet{
			Name:                   name,
			SubnetID:               ptr.StringFromPointer(s.SubnetId),
			AccountID:              payload.AccountID,
			SubnetArn:              ptr.StringFromPointer(s.SubnetArn),
			VpcID:                  ptr.StringFromPointer(s.VpcId),
			State:                  string(s.State),
			AZ:                     ptr.StringFromPointer(s.AvailabilityZone),
			AzID:                   ptr.StringFromPointer(s.AvailabilityZoneId),
			AvailableIPv4Addresses: int(ptr.Value(s.AvailableIpAddressCount, 0)),
			IPv4CIDR:               ptr.StringFromPointer(s.CidrBlock),
			IPv6CIDR:               "", // TODO: fetch IPv6 CIDR
		}
		subnets = append(subnets, item)
	}

	if len(subnets) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&subnets).
		On("CONFLICT (subnet_id, account_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("subnet_arn = EXCLUDED.subnet_arn").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("state = EXCLUDED.state").
		Set("az = EXCLUDED.az").
		Set("az_id = EXCLUDED.az_id").
		Set("available_ipv4_addresses = EXCLUDED.available_ipv4_addresses").
		Set("ipv4_cidr = EXCLUDED.ipv4_cidr").
		Set("ipv6_cidr = EXCLUDED.ipv6_cidr").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert aws subnets into db",
			"region", payload.Region,
			"account_id", payload.AccountID,
			"reason", err,
		)

		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info(
		"populated aws subnets",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	// Emit metrics by grouping the subnets by VPC
	groups := utils.GroupBy(subnets, func(item models.Subnet) string {
		return item.VpcID
	})
	for vpcID, items := range groups {
		metric := prometheus.MustNewConstMetric(
			subnetsDesc,
			prometheus.GaugeValue,
			float64(len(items)),
			payload.AccountID,
			payload.Region,
			vpcID,
		)
		key := metrics.Key(TaskCollectSubnets, payload.AccountID, payload.Region, vpcID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}

	return nil
}
