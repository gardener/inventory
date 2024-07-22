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
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_VPC_TYPE        = "aws:task:collect-vpcs"
	AWS_COLLECT_VPC_REGION_TYPE = "aws:task:collect-vpcs-region"
)

// CollectVpcsPayload is the payload for collecting VPCs for a given AWS Region.
type CollectVpcsPayload struct {
	Region string `json:"region"`
}

// NewCollectVpcsTask creates a new task for collecting all VPCs for all
// Regions.
func NewCollectVpcsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_VPC_TYPE, nil)
}

// HandleCollectVpcsTask handles the task, which collects all VPCs for all known
// AWS Regions.
func HandleCollectVpcsTask(ctx context.Context, t *asynq.Task) error {
	return collectVpcs(ctx)
}

func collectVpcs(ctx context.Context) error {
	slog.Info("Collecting AWS VPCs")
	regions := make([]models.Region, 0)
	err := db.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "reason", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		vpcsTask, err := NewCollectVpcsForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := asynqclient.Client.Enqueue(vpcsTask)
		if err != nil {
			slog.Error(
				"could not enqueue task",
				"type", vpcsTask.Type(),
				"region", r.Name,
				"reason", err,
			)
			continue
		}

		slog.Info(
			"enqueued task",
			"type", vpcsTask.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
		)
	}
	return nil
}

// NewCollectVpcsForRegionTask creates a new task for collecting VPCs for a
// given region.
func NewCollectVpcsForRegionTask(region string) (*asynq.Task, error) {
	if region == "" {
		return nil, ErrMissingRegion
	}

	payload, err := json.Marshal(CollectVpcsPayload{Region: region})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(AWS_COLLECT_VPC_REGION_TYPE, payload), nil
}

// HandleCollectVpcsForRegionTask handles the task for collecting VPCs for a
// given AWS Region.
func HandleCollectVpcsForRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectVpcsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return collectVpcsForRegion(ctx, p.Region)
}

func collectVpcsForRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS VPCs", "region", region)
	paginator := ec2.NewDescribeVpcsPaginator(
		clients.EC2,
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
				o.Region = region
			},
		)
		if err != nil {
			slog.Error("could not describe VPCs", "region", region, "reason", err)
			return err
		}
		items = append(items, page.Vpcs...)
	}

	vpcs := make([]models.VPC, 0, len(items))
	for _, vpc := range items {
		name := utils.FetchTag(vpc.Tags, "Name")
		vpcModel := models.VPC{
			Name:       name,
			VpcID:      strings.StringFromPointer(vpc.VpcId),
			State:      string(vpc.State),
			IPv4CIDR:   strings.StringFromPointer(vpc.CidrBlock),
			IPv6CIDR:   "", //TODO: fetch IPv6 CIDR
			IsDefault:  *vpc.IsDefault,
			OwnerID:    strings.StringFromPointer(vpc.OwnerId),
			RegionName: region,
		}
		vpcs = append(vpcs, vpcModel)
	}

	if len(vpcs) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&vpcs).
		On("CONFLICT (vpc_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("state = EXCLUDED.state").
		Set("ipv4_cidr = EXCLUDED.ipv4_cidr").
		Set("ipv6_cidr = EXCLUDED.ipv6_cidr").
		Set("is_default = EXCLUDED.is_default").
		Set("owner_id = EXCLUDED.owner_id").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert VPCs into db", "region", region, "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws vpcs", "region", region, "count", count)

	return nil
}
