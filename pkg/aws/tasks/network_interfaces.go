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
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	awsclient "github.com/gardener/inventory/pkg/clients/aws"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/utils/ptr"
	"github.com/gardener/inventory/pkg/utils/strings"
)

const (
	TaskCollectNetworkInterfaces          = "aws:task:collect-net-interfaces"
	TaskCollectNetworkInterfacesForRegion = "aws:task:collect-net-interfaces-region"
)

// CollectNetworkInterfacesPayload represents the payload for collecting AWS
// Elastic Network Interfaces (ENI).
type CollectNetworkInterfacesPayload struct {
	// Region specifies the region from which to collect.
	Region string
}

// NetCollectNetworkInterfacesTask creates a new [asynq.Task], which triggers
// collection of AWS Elastic Network Interfaces (ENI) for all known regions.
func NewCollectNetworkInterfacesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectNetworkInterfaces, nil)
}

// NewCollectNetworkInterfacesForRegionTask creates a new [asynq.Task] for
// collecting AWS Elastic Network Interfaces (ENI) from the specified region.
func NewCollectNetworkInterfacesForRegionTask(region string) (*asynq.Task, error) {
	if region == "" {
		return nil, ErrMissingRegion
	}

	payload := CollectNetworkInterfacesPayload{
		Region: region,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	task := asynq.NewTask(TaskCollectNetworkInterfacesForRegion, data)

	return task, nil
}

// HandleCollectNetworkInterfacesTask handles the task for collecting AWS
// Elastic Network Interfaces (ENI).
func HandleCollectNetworkInterfacesTask(ctx context.Context, t *asynq.Task) error {
	// Trigger collection for each known region
	regions := make([]models.Region, 0)
	if err := db.DB.NewSelect().Model(&regions).Scan(ctx); err != nil {
		slog.Error("could not select regions from db", "reason", err)
		return err
	}

	for _, r := range regions {
		task, err := NewCollectNetworkInterfacesForRegionTask(r.Name)
		if err != nil {
			slog.Error("failed to create task", "reason", err)
			continue
		}

		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			slog.Error(
				"could not enqueue task",
				"type", info.Type,
				"region", r.Name,
				"reason", err,
			)
			continue
		}

		slog.Info(
			"enqueued task",
			"type", info.Type,
			"id", info.ID,
			"queue", info.Queue,
			"region", r.Name,
		)
	}

	return nil
}

// HandleCollectNetworkInterfacesForRegionTask handles the task for collecting
// AWS Elastic Network Interfaces (ENI) from a specified region.
func HandleCollectNetworkInterfacesForRegionTask(ctx context.Context, t *asynq.Task) error {
	var payload CollectNetworkInterfacesPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	slog.Info("collecting AWS Elastic Network Interfaces", "region", payload.Region)
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(
		awsclient.EC2,
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
			slog.Error("could not describe network interfaces", "region", payload.Region, "reason", err)
			return err
		}

		items = append(items, page.NetworkInterfaces...)
	}

	// Create model instances from the collected data
	networkInterfaces := make([]models.NetworkInterface, 0, len(items))
	for _, item := range items {
		netInterface := models.NetworkInterface{
			RegionName:       payload.Region,
			AZ:               strings.StringFromPointer(item.AvailabilityZone),
			Description:      strings.StringFromPointer(item.Description),
			InterfaceType:    string(item.InterfaceType),
			MacAddress:       strings.StringFromPointer(item.MacAddress),
			InterfaceID:      strings.StringFromPointer(item.NetworkInterfaceId),
			OwnerID:          strings.StringFromPointer(item.OwnerId),
			PrivateDNSName:   strings.StringFromPointer(item.PrivateDnsName),
			PrivateIPAddress: strings.StringFromPointer(item.PrivateIpAddress),
			RequesterID:      strings.StringFromPointer(item.RequesterId),
			RequesterManaged: ptr.Value(item.RequesterManaged, false),
			SourceDestCheck:  ptr.Value(item.SourceDestCheck, false),
			Status:           string(item.Status),
			SubnetID:         strings.StringFromPointer(item.SubnetId),
			VpcID:            strings.StringFromPointer(item.VpcId),
		}

		// Association
		if item.Association != nil {
			netInterface.AllocationID = strings.StringFromPointer(item.Association.AllocationId)
			netInterface.AssociationID = strings.StringFromPointer(item.Association.AssociationId)
			netInterface.IPOwnerID = strings.StringFromPointer(item.Association.IpOwnerId)
			netInterface.PublicDNSName = strings.StringFromPointer(item.Association.PublicDnsName)
			netInterface.PublicIPAddress = strings.StringFromPointer(item.Association.PublicIp)
		}

		// Attachment
		if item.Attachment != nil {
			netInterface.AttachmentID = strings.StringFromPointer(item.Attachment.AttachmentId)
			netInterface.DeleteOnTermination = ptr.Value(item.Attachment.DeleteOnTermination, false)
			netInterface.DeviceIndex = int(ptr.Value(item.Attachment.DeviceIndex, 0))
			netInterface.InstanceID = strings.StringFromPointer(item.Attachment.InstanceId)
			netInterface.InstanceOwnerID = strings.StringFromPointer(item.Attachment.InstanceOwnerId)
			netInterface.AttachmentStatus = string(item.Attachment.Status)
		}

		networkInterfaces = append(networkInterfaces, netInterface)
	}

	if len(networkInterfaces) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&networkInterfaces).
		On("CONFLICT (interface_id) DO UPDATE").
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
		slog.Error("could not insert network interfaces into db", "region", payload.Region, "reason", err)
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	slog.Info("populated aws network interfaces", "region", payload.Region, "count", count)

	return nil
}
