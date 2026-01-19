// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	// TaskCollectInstances is the name of the task for collecting AWS EC2
	// Instances.
	TaskCollectInstances = "aws:task:collect-instances"
)

// CollectInstancesPayload represents the payload for collecting EC2 Instances.
type CollectInstancesPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectInstancesTask creates a new [asynq.Task] for collecting EC2
// Instances, without specifying a payload.
func NewCollectInstancesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectInstances, nil)
}

// HandleCollectInstancesTask handles the task for collecting EC2 Instances.
func HandleCollectInstancesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting EC2 Instances from all known regions and accounts.
	data := t.Payload()
	if data == nil {
		return enqueueCollectInstances(ctx)
	}

	var payload CollectInstancesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	return collectInstances(ctx, payload)
}

// enqueueCollectInstances enqueues tasks for collecting AWS EC2 Instances from
// all known AWS Regions by creating a payload with the respective region and
// Account ID.
func enqueueCollectInstances(ctx context.Context) error {
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Enqueue task for each known region and account id
	for _, r := range regions {
		if !awsclients.EC2Clientset.Exists(r.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", r.Name,
				"account_id", r.AccountID,
			)

			continue
		}

		payload := CollectInstancesPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS EC2 instances",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectInstances, data)
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

// collectInstances collects the AWS EC2 instances from the specified region,
// using the client associated with the AccountID in the given payload.
func collectInstances(ctx context.Context, payload CollectInstancesPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	logger := asynqutils.GetLogger(ctx)

	logger.Info(
		"collecting AWS instances ",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	paginator := ec2.NewDescribeInstancesPaginator(
		client.Client,
		&ec2.DescribeInstancesInput{},
		func(params *ec2.DescribeInstancesPaginatorOptions) {
			params.Limit = int32(constants.PageSize)
			params.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.Instance, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *ec2.Options) {
				o.Region = payload.Region
			},
		)
		if err != nil {
			logger.Error(
				"could not describe instances",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return awsutils.MaybeSkipRetry(err)
		}

		for _, reservation := range page.Reservations {
			items = append(items, reservation.Instances...)
		}
	}

	instances := make([]models.Instance, 0, len(items))
	for _, instance := range items {
		name := awsutils.FetchTag(instance.Tags, "Name")
		item := models.Instance{
			Name:         name,
			Arch:         string(instance.Architecture),
			InstanceID:   ptr.StringFromPointer(instance.InstanceId),
			AccountID:    payload.AccountID,
			InstanceType: string(instance.InstanceType),
			State:        string(instance.State.Name),
			SubnetID:     ptr.StringFromPointer(instance.SubnetId),
			VpcID:        ptr.StringFromPointer(instance.VpcId),
			Platform:     ptr.StringFromPointer(instance.PlatformDetails),
			RegionName:   payload.Region,
			ImageID:      ptr.StringFromPointer(instance.ImageId),
			LaunchTime:   ptr.Value(instance.LaunchTime, time.Time{}),
		}
		instances = append(instances, item)
	}

	if len(instances) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&instances).
		On("CONFLICT (instance_id, account_id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("arch = EXCLUDED.arch").
		Set("instance_type = EXCLUDED.instance_type").
		Set("state = EXCLUDED.state").
		Set("subnet_id = EXCLUDED.subnet_id").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("platform = EXCLUDED.platform").
		Set("region_name = EXCLUDED.region_name").
		Set("image_id = EXCLUDED.image_id").
		Set("launch_time = EXCLUDED.launch_time").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert instances into db",
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
		"populated aws instances",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	// Emit metrics by grouping the instances by VPC
	groups := utils.GroupBy(instances, func(item models.Instance) string {
		return item.VpcID
	})
	for vpcID, items := range groups {
		// An empty VPC ID would capture instances in terminating state,
		// so we simply exclude these.
		if vpcID == "" {
			continue
		}

		metric := prometheus.MustNewConstMetric(
			instancesDesc,
			prometheus.GaugeValue,
			float64(len(items)),
			payload.AccountID,
			payload.Region,
			vpcID,
		)
		key := metrics.Key(TaskCollectInstances, payload.AccountID, payload.Region, vpcID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}

	return nil
}
