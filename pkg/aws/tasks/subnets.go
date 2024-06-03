package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	awscl "github.com/gardener/inventory/pkg/aws/clients"
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

func NewCollectSubnetsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_SUBNETS_TYPE, nil)
}

func NewCollectSubnetsRegionTask(region string) *asynq.Task {
	if region == "" {
		return nil
	}
	payload, err := json.Marshal(CollectSubnetsPayload{Region: region})
	if err != nil {
		return nil
	}
	return asynq.NewTask(AWS_COLLECT_SUBNETS_REGION_TYPE, payload)
}

func HandleCollectSubnetsRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectSubnetsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	return collectSubnetsRegion(ctx, p.Region)
}

func collectSubnetsRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS subnets", "region", region)

	subnetsOutput, err := awscl.Ec2.DescribeSubnets(ctx,
		&ec2.DescribeSubnetsInput{},
		func(o *ec2.Options) {
			o.Region = region
		},
	)

	if err != nil {
		slog.Error("could not describe subnets", "err", err)
		return err
	}

	subnets := make([]models.Subnet, 0, len(subnetsOutput.Subnets))
	for _, s := range subnetsOutput.Subnets {
		name := utils.FetchTag(s.Tags, "Name")
		slog.Info("Subnet", "name", name)
		modelSubnet := models.Subnet{
			Name:                   name,
			SubnetID:               strings.StringFromPointer(s.SubnetId),
			VpcID:                  strings.StringFromPointer(s.VpcId),
			State:                  string(s.State),
			AZ:                     strings.StringFromPointer(s.AvailabilityZone),
			AzID:                   strings.StringFromPointer(s.AvailabilityZoneId),
			AvailableIPv4Addresses: int(*s.AvailableIpAddressCount),
			IPv4CIDR:               strings.StringFromPointer(s.CidrBlock), //TODO: this can be nil ? and cause panic
			IPv6CIDR:               "",                                     //TODO: fetch IPv6 CIDR
		}
		subnets = append(subnets, modelSubnet)

	}

	if len(subnets) == 0 {
		return nil
	}
	_, err = clients.Db.NewInsert().
		Model(&subnets).
		On("CONFLICT (subnet_id) DO UPDATE").
		Set("name = EXCLUDED.name").
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
		slog.Error("could not insert Subnets into db", "err", err)
		return err
	}

	return nil
}

func HandleCollectSubnetsTask(ctx context.Context, t *asynq.Task) error {
	return collectSubnets(ctx)
}

func collectSubnets(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	err := clients.Db.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "err", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		subnetTask := NewCollectSubnetsRegionTask(r.Name)
		info, err := clients.Client.Enqueue(subnetTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", subnetTask.Type(), "err", err)
			continue
		}
		slog.Info("enqueued task", "type", subnetTask.Type(), "id", info.ID, "queue", info.Queue)
	}
	return nil
}
