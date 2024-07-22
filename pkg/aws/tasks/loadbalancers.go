// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"fmt"
	"log/slog"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/hibiken/asynq"
	"gopkg.in/yaml.v3"

	"github.com/gardener/inventory/pkg/aws/constants"
	"github.com/gardener/inventory/pkg/aws/models"
	"github.com/gardener/inventory/pkg/clients"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	AWS_COLLECT_LOADBALANCERS_TYPE        = "aws:task:collect-lbs"
	AWS_COLLECT_LOADBALANCERS_REGION_TYPE = "aws:task:collect-lbs-region"
)

// CollectLoadBalancersForRegionPayload is the payload needed for aws:task:collect-lbs-region
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
func NewCollectLoadBalancersForRegionTask(region string) (*asynq.Task, error) {
	if region == "" {
		return nil, ErrMissingRegion
	}

	payload, err := yaml.Marshal(CollectLoadBalancersForRegionPayload{Region: region})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(AWS_COLLECT_LOADBALANCERS_REGION_TYPE, payload), nil
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

	return collectLoadBalancersForRegion(ctx, payload.Region)
}

func collectLoadBalancersForRegion(ctx context.Context, region string) error {
	slog.Info("Collecting AWS LoadBalancers", "region", region)

	pageSize := int32(constants.PageSize)
	paginator := elb.NewDescribeLoadBalancersPaginator(
		clients.ELB,
		&elb.DescribeLoadBalancersInput{PageSize: &pageSize},
		func(params *elb.DescribeLoadBalancersPaginatorOptions) {
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.LoadBalancer, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *elb.Options) {
				o.Region = region
			},
		)
		if err != nil {
			slog.Error("could not describe load balancers", "region", region, "reason", err)
			return err
		}
		items = append(items, page.LoadBalancers...)
	}

	lbs := make([]models.LoadBalancer, 0, len(items))
	for _, lb := range items {
		modelLb := models.LoadBalancer{
			ARN:                   strings.StringFromPointer(lb.LoadBalancerArn),
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

	if len(lbs) == 0 {
		return nil
	}

	out, err := clients.DB.NewInsert().
		Model(&lbs).
		On("CONFLICT (arn) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("dns_name = EXCLUDED.dns_name").
		Set("ip_address_type = EXCLUDED.ip_address_type").
		Set("canonical_hosted_zone_id = EXCLUDED.canonical_hosted_zone_id").
		Set("state = EXCLUDED.state").
		Set("scheme = EXCLUDED.scheme").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		slog.Error("could not insert load balancer into db", "region", region, "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws load balancers", "region", region, "count", count)

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
		slog.Error("could not select regions from db", "reason", err)
		return err
	}

	for _, r := range regions {
		// Trigger Asynq task for each region
		lbTask, err := NewCollectLoadBalancersForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := asynqclient.Client.Enqueue(lbTask)
		if err != nil {
			slog.Error(
				"could not enqueue task",
				"type", lbTask.Type(),
				"region", r.Name,
				"reason", err,
			)
			continue
		}

		slog.Info(
			"enqueued task",
			"type", lbTask.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
		)
	}

	return nil
}
