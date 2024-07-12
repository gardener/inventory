package tasks

import (
	"context"
	"fmt"
	"log/slog"

	"gopkg.in/yaml.v3"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_LOADBALANCERS_TYPE        = "aws:task:collect-lbs"
	AWS_COLLECT_LOADBALANCERS_REGION_TYPE = "aws:task:collect-lbs-region"
)

type CollectLoadBalancersForRegionPayload struct {
	Region string `yaml:"region"`
}

// NewCollectLoadBalancersnTask creates a new task for collecting load balancers from
// all AWS Regions.
func NewCollectLoadBalancersTask() *asynq.Task {
	return asynq.NewTask(AWS_COLLECT_LOADBALANCERS_TYPE, nil)
}

// NewCollectLoadbalancersForRegionTask creates a new task for collecting load balancers
// for a given AWS Region.
func NewCollectLoadBalancersForRegionTask(payload CollectLoadBalancersForRegionPayload) (*asynq.Task, error) {
	if payload.Region == "" {
		return nil, ErrMissingRegion
	}

	rawPayload, err := yaml.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(AWS_COLLECT_LOADBALANCERS_REGION_TYPE, rawPayload), nil
}

// HandleCollectLoadBalancersForRegionTask collects load balancers for a specific Region.
func HandleCollectLoadBalancersForRegionTask(ctx context.Context, t *asynq.Task) error {
	var payload CollectLoadBalancersForRegionPayload
	if err := yaml.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("yaml.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	if payload.Region == "" {
		return ErrMissingRegion
	}

	return collectLoadBalancersForRegion(ctx, payload)
}

func collectLoadBalancersForRegion(ctx context.Context, payload CollectLoadBalancersForRegionPayload) error {
	region := payload.Region

	slog.Info("Collecting AWS LoadBalancers", "region", region)

	lbOutput, err := clients.ELB.DescribeLoadBalancers(ctx,
		&elb.DescribeLoadBalancersInput{},
		func(o *elb.Options) {
			o.Region = region
		},
	)

	if err != nil {
		slog.Error("could not describe load balancers", "err", err)
		return err
	}

	count := len(lbOutput.LoadBalancers)
	slog.Info("found load balancers", "count", count, "region", region)
	if count == 0 {
		return nil
	}

	lbs := make([]models.LoadBalancer, 0, count)

	for _, lb := range lbOutput.LoadBalancers {
		modelLb := models.LoadBalancer{
			LbArn:                 strings.StringFromPointer(lb.LoadBalancerArn),
			Name:                  strings.StringFromPointer(lb.LoadBalancerName),
			DNSName:               strings.StringFromPointer(lb.DNSName),
			IpAddressType:         string(lb.IpAddressType),
			CanonicalHostedZoneId: strings.StringFromPointer(lb.CanonicalHostedZoneId),
			State:                 string(lb.State.Code),
			Scheme:                string(lb.Scheme),
			VpcID:                 strings.StringFromPointer(lb.VpcId),
			RegionName:            region,
		}

		lbs = append(lbs, modelLb)
	}

	_, err = clients.DB.NewInsert().
		Model(&lbs).
		On("CONFLICT (lb_arn) DO UPDATE").
		Set("lb_arn = EXCLUDED.lb_arn").
		Set("dns_name = EXCLUDED.dns_name").
		Set("ip_address_type = EXCLUDED.ip_address_type").
		Set("canonical_hosted_zone_id = EXCLUDED.canonical_hosted_zone_id").
		Set("state = EXCLUDED.state").
		Set("scheme = EXCLUDED.scheme").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("region_name = EXCLUDED.region_name").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert load balancer into db", "err", err)
		return err
	}

	return nil
}

// HandleCollectLoadBalancersTask collects load balancers for all known regions
func HandleCollectLoadBalancersTask(ctx context.Context, t *asynq.Task) error {
	return collectLoadBalancers(ctx)
}

func collectLoadBalancers(ctx context.Context) error {
	// Collect regions from Db
	regions := make([]models.Region, 0)
	if err := clients.DB.NewSelect().Model(&regions).Scan(ctx); err != nil {
		slog.Error("could not select regions from db", "err", err)
		return err
	}

	for _, r := range regions {
		// Trigger Asynq task for each region
		collectLoadBalancersForRegionPayload := CollectLoadBalancersForRegionPayload{
			Region: r.Name,
		}

		lbTask, err := NewCollectLoadBalancersForRegionTask(collectLoadBalancersForRegionPayload)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := clients.Client.Enqueue(lbTask)
		if err != nil {
			slog.Error("could not enqueue task", "type", lbTask.Type(), "err", err)
			continue
		}

		slog.Info("enqueued task", "type", lbTask.Type(), "id", info.ID, "queue", info.Queue)
	}

	return nil
}
