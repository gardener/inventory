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

// TaskCollectVPCs is the name of the task for collecting Azure VPCs.
const TaskCollectVPCs = "az:task:collect-vpcs"

// CollectVPCsPayload is the payload used for collecting Azure
// VPCs.
type CollectVPCsPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`
}

// NewCollectVPCsTask creates a new [asynq.Task] for collecting Azure
// VPCs without specifying a payload.
func NewCollectVPCsTask() *asynq.Task {
	return asynq.NewTask(TaskCollectVPCs, nil)
}

// HandleCollectVPCsTask is the handler, which collects Azure
// VPCs.
func HandleCollectVPCsTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known resource groups.
	data := t.Payload()
	if data == nil {
		return enqueueCollectVPCs(ctx)
	}

	var payload CollectVPCsPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}

	return collectVPCs(ctx, payload)
}

// enqueueVPCs enqueues tasks for collecting Azure VPCs
// for known Resource Groups.
func enqueueCollectVPCs(ctx context.Context) error {
	resourceGroups, err := azureutils.GetResourceGroupsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	for _, rg := range resourceGroups {
		if !azureclients.VirtualNetworksClientset.Exists(rg.SubscriptionID) {
			logger.Warn(
				"Azure VPCs client not found",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
			)
			continue
		}

		payload := CollectVPCsPayload{
			SubscriptionID: rg.SubscriptionID,
			ResourceGroup:  rg.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure VPCs",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
				"reason", err,
			)
			continue
		}
		task := asynq.NewTask(TaskCollectVPCs, data)
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

// collectVPCs collects the Azure VPCs from the
// subscription and resource group specified in the payload.
func collectVPCs(ctx context.Context, payload CollectVPCsPayload) error {
	client, ok := azureclients.VirtualNetworksClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure VPCs",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
	)

	items := make([]models.VPC, 0)
	pager := client.Client.NewListPager(
		payload.ResourceGroup,
		&armnetwork.VirtualNetworksClientListOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure VPCs",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"reason", err,
			)
			return azureutils.MaybeSkipRetry(err)
		}

		for _, vpc := range page.Value {
			var provisioningState armnetwork.ProvisioningState
			var encryptionEnabled *bool
			var vmProtectionEnabled *bool

			if vpc.Properties != nil {
				provisioningState = ptr.Value(vpc.Properties.ProvisioningState, armnetwork.ProvisioningState(""))
				if vpc.Properties.Encryption != nil {
					encryptionEnabled = vpc.Properties.Encryption.Enabled
				}
				vmProtectionEnabled = vpc.Properties.EnableVMProtection
			}

			item := models.VPC{
				Name:                ptr.Value(vpc.Name, ""),
				SubscriptionID:      payload.SubscriptionID,
				ResourceGroupName:   payload.ResourceGroup,
				Location:            ptr.Value(vpc.Location, ""),
				ProvisioningState:   string(provisioningState),
				EncryptionEnabled:   ptr.Value(encryptionEnabled, false),
				VMProtectionEnabled: ptr.Value(vmProtectionEnabled, false),
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
		Set("encryption_enabled = EXCLUDED.encryption_enabled").
		Set("vm_protection_enabled = EXCLUDED.vm_protection_enabled").
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

	logger.Info("populated azure vpcs", "count", count)

	return nil
}
