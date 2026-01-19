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
	// TaskCollectNetworkInterfaces is the name of the task for collecting
	// AWS ENIs.
	TaskCollectNetworkInterfaces = "aws:task:collect-net-interfaces"
)

// CollectNetworkInterfacesPayload represents the payload for collecting AWS
// Elastic Network Interfaces (ENI).
type CollectNetworkInterfacesPayload struct {
	// Region specifies the region from which to collect.
	Region string `json:"region" yaml:"region"`

	// AccountID specifies the AWS Account ID, which is associated with a
	// registered client.
	AccountID string `json:"account_id" yaml:"account_id"`
}

// NewCollectNetworkInterfacesTask creates a new [asynq.Task] for collecting AWS
// ENIs, without specifying a payload.
func NewCollectNetworkInterfacesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectNetworkInterfaces, nil)
}

// HandleCollectNetworkInterfacesTask handles the task for collecting AWS
// Elastic Network Interfaces (ENI).
func HandleCollectNetworkInterfacesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue tasks for
	// collecting ENIs from all known regions and their respective accounts.
	data := t.Payload()
	if data == nil {
		return enqueueCollectENIs(ctx)
	}

	var payload CollectNetworkInterfacesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.AccountID == "" {
		return asynqutils.SkipRetry(ErrNoAccountID)
	}

	if payload.Region == "" {
		return asynqutils.SkipRetry(ErrNoRegion)
	}

	return collectENIs(ctx, payload)
}

// enqueueCollectENIs enqueues tasks for collecting AWS ENIs for the known
// regions and accounts.
func enqueueCollectENIs(ctx context.Context) error {
	regions, err := awsutils.GetRegionsFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get regions: %w", err)
	}

	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)

	// Enqueue ENI collection for each region
	for _, r := range regions {
		if !awsclients.EC2Clientset.Exists(r.AccountID) {
			logger.Warn(
				"AWS client not found",
				"region", r.Name,
				"account_id", r.AccountID,
			)

			continue
		}

		payload := CollectNetworkInterfacesPayload{
			Region:    r.Name,
			AccountID: r.AccountID,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for AWS ENIs",
				"region", r.Name,
				"account_id", r.AccountID,
				"reason", err,
			)

			continue
		}

		task := asynq.NewTask(TaskCollectNetworkInterfaces, data)
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

// collectENIs collects the AWS ENIs from the specified region using the client
// associated with the given AccountID from the payload.
func collectENIs(ctx context.Context, payload CollectNetworkInterfacesPayload) error {
	client, ok := awsclients.EC2Clientset.Get(payload.AccountID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.AccountID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting AWS ENIs",
		"region", payload.Region,
		"account_id", payload.AccountID,
	)

	paginator := ec2.NewDescribeNetworkInterfacesPaginator(
		client.Client,
		&ec2.DescribeNetworkInterfacesInput{},
		func(opts *ec2.DescribeNetworkInterfacesPaginatorOptions) {
			opts.Limit = int32(constants.PageSize)
			opts.StopOnDuplicateToken = true
		},
	)

	// Fetch items from all pages
	items := make([]types.NetworkInterface, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(
			ctx,
			func(o *ec2.Options) {
				o.Region = payload.Region
			},
		)

		if err != nil {
			logger.Error(
				"could not describe network interfaces",
				"region", payload.Region,
				"account_id", payload.AccountID,
				"reason", err,
			)

			return awsutils.MaybeSkipRetry(err)
		}
		items = append(items, page.NetworkInterfaces...)
	}

	// Create model instances from the collected data
	networkInterfaces := make([]models.NetworkInterface, 0, len(items))
	for _, item := range items {
		netInterface := models.NetworkInterface{
			RegionName:       payload.Region,
			AZ:               ptr.StringFromPointer(item.AvailabilityZone),
			Description:      ptr.StringFromPointer(item.Description),
			InterfaceType:    string(item.InterfaceType),
			AccountID:        payload.AccountID,
			MacAddress:       ptr.StringFromPointer(item.MacAddress),
			InterfaceID:      ptr.StringFromPointer(item.NetworkInterfaceId),
			OwnerID:          ptr.StringFromPointer(item.OwnerId),
			PrivateDNSName:   ptr.StringFromPointer(item.PrivateDnsName),
			PrivateIPAddress: ptr.StringFromPointer(item.PrivateIpAddress),
			RequesterID:      ptr.StringFromPointer(item.RequesterId),
			RequesterManaged: ptr.Value(item.RequesterManaged, false),
			SourceDestCheck:  ptr.Value(item.SourceDestCheck, false),
			Status:           string(item.Status),
			SubnetID:         ptr.StringFromPointer(item.SubnetId),
			VpcID:            ptr.StringFromPointer(item.VpcId),
		}

		// Association
		if item.Association != nil {
			netInterface.AllocationID = ptr.StringFromPointer(item.Association.AllocationId)
			netInterface.AssociationID = ptr.StringFromPointer(item.Association.AssociationId)
			netInterface.IPOwnerID = ptr.StringFromPointer(item.Association.IpOwnerId)
			netInterface.PublicDNSName = ptr.StringFromPointer(item.Association.PublicDnsName)
			netInterface.PublicIPAddress = ptr.StringFromPointer(item.Association.PublicIp)
		}

		// Attachment
		if item.Attachment != nil {
			netInterface.AttachmentID = ptr.StringFromPointer(item.Attachment.AttachmentId)
			netInterface.DeleteOnTermination = ptr.Value(item.Attachment.DeleteOnTermination, false)
			netInterface.DeviceIndex = int(ptr.Value(item.Attachment.DeviceIndex, 0))
			netInterface.InstanceID = ptr.StringFromPointer(item.Attachment.InstanceId)
			netInterface.InstanceOwnerID = ptr.StringFromPointer(item.Attachment.InstanceOwnerId)
			netInterface.AttachmentStatus = string(item.Attachment.Status)
		}

		networkInterfaces = append(networkInterfaces, netInterface)
	}

	if len(networkInterfaces) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&networkInterfaces).
		On("CONFLICT (interface_id, account_id) DO UPDATE").
		Set("az = EXCLUDED.az").
		Set("description = EXCLUDED.description").
		Set("interface_type = EXCLUDED.interface_type").
		Set("mac_address = EXCLUDED.mac_address").
		Set("owner_id = EXCLUDED.owner_id").
		Set("private_dns_name = EXCLUDED.private_dns_name").
		Set("private_ip_address = EXCLUDED.private_ip_address").
		Set("requester_id = EXCLUDED.requester_id").
		Set("requester_managed = EXCLUDED.requester_managed").
		Set("src_dst_check = EXCLUDED.src_dst_check").
		Set("status = EXCLUDED.status").
		Set("subnet_id = EXCLUDED.subnet_id").
		Set("vpc_id = EXCLUDED.vpc_id").
		Set("allocation_id = EXCLUDED.allocation_id").
		Set("association_id = EXCLUDED.association_id").
		Set("ip_owner_id = EXCLUDED.ip_owner_id").
		Set("public_dns_name = EXCLUDED.public_dns_name").
		Set("public_ip_address = EXCLUDED.public_ip_address").
		Set("attachment_id = EXCLUDED.attachment_id").
		Set("delete_on_termination = EXCLUDED.delete_on_termination").
		Set("device_index = EXCLUDED.device_index").
		Set("instance_id = EXCLUDED.instance_id").
		Set("instance_owner_id = EXCLUDED.instance_owner_id").
		Set("attachment_status = EXCLUDED.attachment_status").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		logger.Error(
			"could not insert network interfaces into db",
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
		"populated aws network interfaces",
		"region", payload.Region,
		"account_id", payload.AccountID,
		"count", count,
	)

	// Emit metrics
	groups := utils.GroupBy(networkInterfaces, func(item models.NetworkInterface) string {
		return item.VpcID
	})
	for vpcID, items := range groups {
		metric := prometheus.MustNewConstMetric(
			netInterfacesDesc,
			prometheus.GaugeValue,
			float64(len(items)),
			payload.AccountID,
			payload.Region,
			vpcID,
		)
		key := metrics.Key(TaskCollectNetworkInterfaces, payload.AccountID, payload.Region, vpcID)
		metrics.DefaultCollector.AddMetric(key, metric)
	}

	return nil
}
