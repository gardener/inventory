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
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/constants"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/aws/utils"
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_SUBNETS_TYPE        = "aws:task:collect-subnets"
	AWS_COLLECT_SUBNETS_REGION_TYPE = "aws:task:collect-subnets-region"
)

type CollectSubnetsPayload struct {
	Region string `json:"region"`
}

// NewCollectSubnetTask creates a new task for collecting all Subnets from all
// Regions.
func NewCollectSubnetsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_SUBNETS_TYPE, nil)
}

// NewCollectSubnetsForRegionTask creates a task for collecting Subnets from a
// given Region.
func NewCollectSubnetsForRegionTask(region string) (*asynq.Task, error) {
	if region == "" {
		return nil, ErrMissingRegion
	}

	payload, err := json.Marshal(CollectSubnetsPayload{Region: region})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(AWS_COLLECT_SUBNETS_REGION_TYPE, payload), nil
}

// HandleCollectSubnetsForRegionTask collects the Subnets from a specific
// Region, provided as part of the task payload.
func HandleCollectSubnetsForRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectSubnetsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return collectSubnetsForRegion(ctx, p.Region)
}

func collectSubnetsForRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS subnets", "region", region)
	paginator := ec2.NewDescribeSubnetsPaginator(
		clients.EC2,
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
				o.Region = region
			},
		)
		if err != nil {
			slog.Error("could not describe subnets", "region", region, "reason", err)
			return err
		}
		items = append(items, page.Subnets...)
	}

	subnets := make([]models.Subnet, 0, len(items))
	for _, s := range items {
		name := utils.FetchTag(s.Tags, "Name")
		modelSubnet := models.Subnet{
			Name:                   name,
			SubnetID:               strings.StringFromPointer(s.SubnetId),
			SubnetArn:              strings.StringFromPointer(s.SubnetArn),
			VpcID:                  strings.StringFromPointer(s.VpcId),
			State:                  string(s.State),
			AZ:                     strings.StringFromPointer(s.AvailabilityZone),
			AzID:                   strings.StringFromPointer(s.AvailabilityZoneId),
			AvailableIPv4Addresses: int(*s.AvailableIpAddressCount),
			IPv4CIDR:               strings.StringFromPointer(s.CidrBlock),
			IPv6CIDR:               "", //TODO: fetch IPv6 CIDR
		}
		subnets = append(subnets, modelSubnet)
	}

	if len(subnets) == 0 {
		return nil
	}

	out, err := clients.DB.NewInsert().
		Model(&subnets).
		On("CONFLICT (subnet_id) DO UPDATE").
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
		slog.Error("could not insert Subnets into db", "region", region, "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws subnets", "region", region, "count", count)

	return nil
}

// HandleCollectSubnetsTask collects all AWS Subnets from all AWS Regions.
func HandleCollectSubnetsTask(ctx context.Context, t *asynq.Task) error {
	return collectSubnets(ctx)
}

func collectSubnets(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := clients.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "reason", err)
		return err
	}

	for _, r := range regions {
		// Trigger Asynq task for each region
		subnetTask, err := NewCollectSubnetsForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := clients.Client.Enqueue(subnetTask)
		if err != nil {
			slog.Error(
				"could not enqueue task",
				"type", subnetTask.Type(),
				"region", r.Name,
				"reason", err,
			)
			continue
		}

		slog.Info(
			"enqueued task",
			"type", subnetTask.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
		)
	}
	return nil
}
