// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/azure/models"
	azureutils "github.com/gardener/inventory/pkg/azure/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectLoadBalancers is the name of the task for collecting Azure Load
// Balancers.
const TaskCollectLoadBalancers = "az:task:collect-loadbalancers"

// CollectLoadBalancersPayload is the payload used for collecting Azure Load
// Balancers.
type CollectLoadBalancersPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`
}

// NewCollectLoadBalancersTask creates a new [asynq.Task] for collecting Azure
// Load Balancers, without specifying a payload.
func NewCollectLoadBalancersTask() *asynq.Task {
	return asynq.NewTask(TaskCollectLoadBalancers, nil)
}

// HandleCollectLoadBalancersTask is the handler, which collects Azure Load
// Balancers.
func HandleCollectLoadBalancersTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known resource groups.
	data := t.Payload()
	if data == nil {
		return enqueueCollectLoadBalancers(ctx)
	}

	var payload CollectLoadBalancersPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}

	return collectLoadBalancers(ctx, payload)
}

// enqueueCollectLoadBalancers enqueues tasks for collecting Azure Load
// Balancers for the known Resource Groups.
func enqueueCollectLoadBalancers(ctx context.Context) error {
	resourceGroups, err := azureutils.GetResourceGroupsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	for _, rg := range resourceGroups {
		if !azureclients.LoadBalancersClientset.Exists(rg.SubscriptionID) {
			logger.Warn(
				"Azure Load Balancer client not found",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
			)
			continue
		}

		payload := CollectLoadBalancersPayload{
			SubscriptionID: rg.SubscriptionID,
			ResourceGroup:  rg.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure Load Balancers",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
				"reason", err,
			)
			continue
		}
		task := asynq.NewTask(TaskCollectLoadBalancers, data)
		info, err := asynqclient.Client.Enqueue(task)
		if err != nil {
			logger.Error(
				"failed to enqueue task",
				"type", task.Type(),
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
				"reason", err,
			)
			continue
		}

		logger.Info(
			"enqueued task",
			"type", task.Type(),
			"id", info.ID,
			"queue", info.Queue,
			"subscription_id", rg.SubscriptionID,
			"resource_group", rg.Name,
		)
	}

	return nil
}

// collectLoadBalancers collects the Azure Load Balancers from the subscription
// and resource group specified in the payload.
func collectLoadBalancers(ctx context.Context, payload CollectLoadBalancersPayload) error {
	client, ok := azureclients.LoadBalancersClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure Load Balancers",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
	)

	items := make([]models.LoadBalancer, 0)
	pager := client.Client.NewListPager(
		payload.ResourceGroup,
		&armnetwork.LoadBalancersClientListOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure Load Balancers",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"reason", err,
			)
			return azureutils.MaybeSkipRetry(err)
		}

		// NOTE: Frontend and Backend configuration for Load Balancers is not
		// collected at the moment, because the Go SDK for Azure does not return
		// results for them. See [1] for more details.
		//
		// [1]: https://github.com/Azure/azure-sdk-for-go/issues/23578
		for _, lb := range page.Value {
			var provisioningState armnetwork.ProvisioningState
			if lb.Properties != nil {
				provisioningState = ptr.Value(lb.Properties.ProvisioningState, armnetwork.ProvisioningState(""))
			}
			var skuName armnetwork.LoadBalancerSKUName
			var skuTier armnetwork.LoadBalancerSKUTier
			if lb.SKU != nil {
				skuName = ptr.Value(lb.SKU.Name, armnetwork.LoadBalancerSKUName(""))
				skuTier = ptr.Value(lb.SKU.Tier, armnetwork.LoadBalancerSKUTier(""))

			}
			item := models.LoadBalancer{
				Name:              ptr.Value(lb.Name, ""),
				SubscriptionID:    payload.SubscriptionID,
				ResourceGroupName: payload.ResourceGroup,
				Location:          ptr.Value(lb.Location, ""),
				ProvisioningState: string(provisioningState),
				SKUName:           string(skuName),
				SKUTier:           string(skuTier),
			}
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil
	}

	out, err := db.DB.NewInsert().
		Model(&items).
		On("CONFLICT (subscription_id, resource_group, name) DO UPDATE").
		Set("location = EXCLUDED.location").
		Set("provisioning_state = EXCLUDED.provisioning_state").
		Set("sku_name = EXCLUDED.sku_name").
		Set("sku_tier = EXCLUDED.sku_tier").
		Set("updated_at = EXCLUDED.updated_at").
		Returning("id").
		Exec(ctx)

	if err != nil {
		return err
	}

	count, err := out.RowsAffected()
	if err != nil {
		return err
	}

	logger.Info("populated azure load balancers", "count", count)

	return nil
}
