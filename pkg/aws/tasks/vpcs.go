package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/hibiken/asynq"

	awsclients "github.com/gardener/inventory/pkg/aws/clients"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/aws/utils"
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_VPC_TYPE        = "aws:task:collect-vpcs"
	AWS_COLLECT_VPC_REGION_TYPE = "aws:task:collect-vpcs-region"
)

type CollectVpcsPayload struct {
	Region string `json:"region"`
}

func NewCollectVpcsTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_VPC_TYPE, nil)
}

func HandleCollectVpcsTask(ctx context.Context, t *asynq.Task) error {
	return collectVpcs(ctx)
}

func collectVpcs(ctx context.Context) error {
	slog.Info("Collecting AWS VPCs")
	regions := make([]models.Region, 0)
	err := clients.Db.NewSelect().Model(&regions).Scan(ctx)
	if err != nil {
		slog.Error("could not select regions from db", "err", err)
		return err
	}
	for _, r := range regions {
		// Trigger Asynq task for each region
		vpcsTask := NewCollectVpcsRegionTask(r.Name)
		info, err := clients.Client.Enqueue(vpcsTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", vpcsTask.Type(), "err", err)
			continue
		}
		slog.Info("enqueued task", "type", vpcsTask.Type(), "id", info.ID, "queue", info.Queue)
	}
	return nil
}

func NewCollectVpcsRegionTask(region string) *asynq.Task {
	if region == "" {
		slog.Info("region is required and cannot be empty")
		return nil
	}
	payload, err := json.Marshal(CollectVpcsPayload{Region: region})
	if err != nil {
		slog.Error("could not marshal payload", "err", err)
		return nil
	}
	return asynq.NewTask(AWS_COLLECT_VPC_REGION_TYPE, payload)
}

func HandleCollectVpcsRegionTask(ctx context.Context, t *asynq.Task) error {
	var p CollectVpcsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return errors.New("json.Unmarshal failed")
	}
	return collectVpcsRegion(ctx, p.Region)
}

func collectVpcsRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS VPCs", "region", region)
	vpcsOutput, err := awsclients.Ec2.DescribeVpcs(ctx,
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

	_, err = clients.Db.NewInsert().
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
