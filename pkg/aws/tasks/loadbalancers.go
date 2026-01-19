// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	v1types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	v2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
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
	// TaskCollectLoadBalancers is the name of the task for collecting AWS
	// Elastic Load Balancers (ELBs).
	TaskCollectLoadBalancers = "aws:task:collect-loadbalancers"
)

// CollectLoadBalancersPayload is the payload, which is used for collecting AWS
// ELBs.
type CollectLoadBalancersPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectLoadBalancersTask creates a new [asynq.Task] for collecting AWS
// ELBs, without specifying a payload.
func NewCollectLoadBalancersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectLoadBalancers, nil)
}

// HandleCollectLoadBalancersTask handles the task for collecting AWS ELBs.
func HandleCollectLoadBalancersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting ELBs from all known regions and their respective accounts.
	data := t.Payload()
	if data == nil {
		return enqueueCollectLoadBalancers(ctx)
	}

	var payload CollectLoadBalancersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	return collectLoadBalancers(ctx, payload)
}

// enqueueCollectLoadBalancers enqueues tasks for collecting the ELBs from all
// known AWS Regions.
func enqueueCollectLoadBalancers(ctx context.Context) error {
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Enqueue ELB collection tasks for each region
	for _, r := range regions {
		payload := CollectLoadBalancersPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS ELBs",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectLoadBalancers, data)
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

// collectLoadBalancers collects the AWS ELBs from the specified region in the
// payload.
func collectLoadBalancers(ctx context.Context, payload CollectLoadBalancersPayload) error {
	logger := asynqutils.GetLogger(ctx)
	if awsclients.ELBClientset.Exists(payload.AccountID) {
		if err := collectELBv1(ctx, payload); err != nil {
			logger.Error(
				"failed to collect ELB v1",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)
		}
	} else {
		logger.Warn(
			"AWS client not found",
			"region", payload.Region,
			"account_id", payload.AccountID,
		)
	}

	if awsclients.ELBv2Clientset.Exists(payload.AccountID) {
		if err := collectELBv2(ctx, payload); err != nil {
			logger.Error(
				"failed to collect ELB v2",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)
		}
	} else {
		logger.Warn(
			"AWS client not found",
			"region", payload.Region,
			"account_id", payload.AccountID,
		)
	}

	return nil
}

// collectELBv2 collects ELB v2 load balancers.
func collectELBv2(ctx context.Context, payload CollectLoadBalancersPayload) error {
	client, ok := awsclients.ELBv2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS ELB v2",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	pageSize := int32(constants.PageSize)
	paginator := elbv2.NewDescribeLoadBalancersPaginator(
		client.Client,
		&elbv2.DescribeLoadBalancersInput{PageSize: &pageSize},
		func(params *elbv2.DescribeLoadBalancersPaginatorOptions) {
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]v2types.LoadBalancer, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *elbv2.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not describe AWS ELB v2",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return awsutils.MaybeSkipRetry(err)
		}
		items = append(items, page.LoadBalancers...)
	}

	lbs := make([]models.LoadBalancer, 0, len(items))
	for _, lb := range items {
		// Get the LoadBalancerID from the last component of the ARN
		arn := ptr.StringFromPointer(lb.LoadBalancerArn)
		arnParts := strings.Split(arn, "/")
		loadBalancerID := arnParts[len(arnParts)-1]

		item := models.LoadBalancer{
			Name:                  ptr.StringFromPointer(lb.LoadBalancerName),
			ARN:                   arn,
			LoadBalancerID:        loadBalancerID,
			DNSName:               ptr.StringFromPointer(lb.DNSName),
			AccountID:             payload.AccountID,
			CanonicalHostedZoneID: ptr.StringFromPointer(lb.CanonicalHostedZoneId),
			State:                 string(lb.State.Code),
			Scheme:                string(lb.Scheme),
			Type:                  string(lb.Type),
			VpcID:                 ptr.StringFromPointer(lb.VpcId),
			RegionName:            payload.Region,
		}
		lbs = append(lbs, item)
	}

	if len(lbs) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&lbs).
		On("CONFLICT (dns_name, account_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("arn = EXCLUDED.arn").
		Set("load_balancer_id = EXCLUDED.load_balancer_id").
		Set("canonical_hosted_zone_id = EXCLUDED.canonical_hosted_zone_id").
		Set("state = EXCLUDED.state").
		Set("scheme = EXCLUDED.scheme").
		Set("type = EXCLUDED.type").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert AWS ELB v2 into db",
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
		"populated AWS ELB v2",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	// Emit metrics by grouping the ELBs by VPC
	groups := utils.GroupBy(lbs, func(item models.LoadBalancer) string {
		return item.VpcID
	})
	for vpcID, items := range groups {
		metric := prometheus.MustNewConstMetric(
			loadBalancersDesc,
			prometheus.GaugeValue,
			float64(len(items)),
			payload.AccountID,
			payload.Region,
			vpcID,
		)
		key := metrics.Key(TaskCollectLoadBalancers, payload.AccountID, payload.Region, vpcID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}

	return nil
}

// collectELBv1 collects ELB v1 (classic) load balancers.
func collectELBv1(ctx context.Context, payload CollectLoadBalancersPayload) error {
	client, ok := awsclients.ELBClientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS ELB v1",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	pageSize := int32(constants.PageSize)
	paginator := elb.NewDescribeLoadBalancersPaginator(
		client.Client,
		&elb.DescribeLoadBalancersInput{PageSize: &pageSize},
		func(params *elb.DescribeLoadBalancersPaginatorOptions) {
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]v1types.LoadBalancerDescription, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *elb.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not describe AWS ELB v1",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return awsutils.MaybeSkipRetry(err)
		}
		items = append(items, page.LoadBalancerDescriptions...)
	}

	lbs := make([]models.LoadBalancer, 0, len(items))
	for _, lb := range items {
		item := models.LoadBalancer{
			Name:                  ptr.StringFromPointer(lb.LoadBalancerName),
			DNSName:               ptr.StringFromPointer(lb.DNSName),
			AccountID:             payload.AccountID,
			CanonicalHostedZoneID: ptr.StringFromPointer(lb.CanonicalHostedZoneNameID),
			Scheme:                ptr.StringFromPointer(lb.Scheme),
			Type:                  constants.LoadBalancerClassicType,
			VpcID:                 ptr.StringFromPointer(lb.VPCId),
			RegionName:            payload.Region,
		}
		lbs = append(lbs, item)
	}

	if len(lbs) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&lbs).
		On("CONFLICT (dns_name, account_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("canonical_hosted_zone_id = EXCLUDED.canonical_hosted_zone_id").
		Set("state = EXCLUDED.state").
		Set("scheme = EXCLUDED.scheme").
		Set("type = EXCLUDED.type").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("region_name = EXCLUDED.region_name").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert AWS ELB v1 into db",
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
		"populated AWS ELB v1",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	return nil
}
