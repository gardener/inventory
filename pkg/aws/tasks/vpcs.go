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
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/aws/utils"
	"github.com/gardener/inventory/pkg/clients"
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
	err := clients.DB.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "err", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		vpcsTask, err := NewCollectVpcsForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := clients.Client.Enqueue(vpcsTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", vpcsTask.Type(), "reason", err)
			continue
		}

		slog.Info("enqueued task", "type", vpcsTask.Type(), "id", info.ID, "queue", info.Queue)
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

	return asynq.NewTask(AWS_COLLECT_VPC_REGION_TYPE, payload), err
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
	vpcsOutput, err := clients.EC2.DescribeVpcs(ctx,
		&ec2.DescribeVpcsInput{},
		func(o *ec2.Options) {
			o.Region = region
		})
	if err != nil {
		slog.Error("could not describe VPCs", "region", region, "err", err)
		return err
	}

	vpcs := make([]models.VPC, 0, len(vpcsOutput.Vpcs))
	for _, vpc := range vpcsOutput.Vpcs {
		name := utils.FetchTag(vpc.Tags, "Name")
		slog.Info("VPC", "name", name, "region", region)
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

	_, err = clients.DB.NewInsert().
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
		slog.Error("could not insert VPCs into db", "region", region, "err", err)
		return err
	}

	return nil
}
