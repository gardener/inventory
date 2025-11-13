// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gardener/inventory/pkg/azure/models"
	azureutils "github.com/gardener/inventory/pkg/azure/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	"github.com/gardener/inventory/pkg/metrics"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectSubnets is the name of the task for collecting Azure
// Subnets.
const TaskCollectSubnets = "az:task:collect-subnets"

// CollectSubnetsPayload is the payload used for collecting Azure
// Subnets.
type CollectSubnetsPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`

	// VPCName specifies from which VPC to collect.
	VPCName string `json:"vpc_name" yaml:"vpc_name"`
}

// NewCollectSubnetsTask creates a new [asynq.Task] for collecting Azure
// Subnets, without specifying a payload.
func NewCollectSubnetsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectSubnets, nil)
}

// HandleCollectSubnetsTask is the handler, which collects Azure
// Subnets.
func HandleCollectSubnetsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known resource groups.
	data := t.Payload()
	if data == nil {
		return enqueueCollectSubnets(ctx)
	}

	var payload CollectSubnetsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}
	if payload.VPCName == "" {
		return asynqutils.SkipRetry(ErrNoVPC)
	}

	return collectSubnets(ctx, payload)
}

// enqueueSubnets enqueues tasks for collecting Azure Subnets for known Resource Groups.
func enqueueCollectSubnets(ctx context.Context) error {
	vpcs, err := azureutils.GetVPCsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)
	for _, vpc := range vpcs {
		if !azureclients.SubnetsClientset.Exists(vpc.SubscriptionID) {
			logger.Warn(
				"Azure Subnets client not found",
				"subscription_id", vpc.SubscriptionID,
				"resource_group", vpc.ResourceGroupName,
			)

			continue
		}

		payload := CollectSubnetsPayload{
			SubscriptionID: vpc.SubscriptionID,
			ResourceGroup:  vpc.ResourceGroupName,
			VPCName:        vpc.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure Subnets",
				"subscription_id", vpc.SubscriptionID,
				"resource_group", vpc.ResourceGroupName,
				"vpc", vpc.Name,
				"reason", err,
			)

			continue
		}
		task := asynq.NewTask(TaskCollectSubnets, data)
		info, err := asynqclient.Client.Enqueue(task, asynq.Queue(queue))
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"subscription_id", vpc.SubscriptionID,
				"resource_group", vpc.ResourceGroupName,
				"vpc", vpc.Name,
				"reason", err,
			)

			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"subscription_id", vpc.SubscriptionID,
			"resource_group", vpc.ResourceGroupName,
			"vpc", vpc.Name,
		)
	}

	return nil
}

// collectSubnets collects the Azure Subnets from the
// subscription, resource group and VPC specified in the payload.
func collectSubnets(ctx context.Context, payload CollectSubnetsPayload) error {
	client, ok := azureclients.SubnetsClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure Subnets",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
		"vpc", payload.VPCName,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			subnetsDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.SubscriptionID,
			payload.ResourceGroup,
			payload.VPCName,
		)
		key := metrics.Key(
			TaskCollectSubnets,
			payload.SubscriptionID,
			payload.ResourceGroup,
			payload.VPCName,
		)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	subnets := make([]models.Subnet, 0)
	pager := client.Client.NewListPager(
		payload.ResourceGroup,
		payload.VPCName,
		&armnetwork.SubnetsClientListOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure Subnets",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"vpc", payload.VPCName,
				"reason", err,
			)

			return azureutils.MaybeSkipRetry(err)
		}

		for _, subnet := range page.Value {
			var provisioningState armnetwork.ProvisioningState
			var addressPrefix string
			var purpose string
			var securityGroup string

			if subnet.Properties != nil {
				provisioningState = ptr.Value(subnet.Properties.ProvisioningState, armnetwork.ProvisioningState(""))
				addressPrefix = ptr.Value(subnet.Properties.AddressPrefix, "")
				purpose = ptr.Value(subnet.Properties.Purpose, "")
				if subnet.Properties.NetworkSecurityGroup != nil {
					securityGroup = ptr.Value(subnet.Properties.NetworkSecurityGroup.Name, "")
				}
			}

			item := models.Subnet{
				Name:              ptr.Value(subnet.Name, ""),
				Type:              ptr.Value(subnet.Type, ""),
				SubscriptionID:    payload.SubscriptionID,
				ResourceGroupName: payload.ResourceGroup,
				ProvisioningState: string(provisioningState),
				VPCName:           payload.VPCName,
				AddressPrefix:     addressPrefix,
				SecurityGroup:     securityGroup,
				Purpose:           purpose,
			}
			subnets = append(subnets, item)
		}
	}

	if len(subnets) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&subnets).
		On("CONFLICT (subscription_id, resource_group, vpc_name, name) DO UPDATE").
		Set("type = EXCLUDED.type").
		Set("provisioning_state = EXCLUDED.provisioning_state").
		Set("address_prefix = EXCLUDED.address_prefix").
		Set("security_group = EXCLUDED.security_group").
		Set("purpose = EXCLUDED.purpose").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err = out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated azure subnets", "count", count)

	return nil
}
