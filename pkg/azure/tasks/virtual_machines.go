// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"time"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/hibiken/asynq"

	"github.com/gardener/inventory/pkg/azure/models"
	azureutils "github.com/gardener/inventory/pkg/azure/utils"
	asynqclient "github.com/gardener/inventory/pkg/clients/asynq"
	azureclients "github.com/gardener/inventory/pkg/clients/azure"
	"github.com/gardener/inventory/pkg/clients/db"
	asynqutils "github.com/gardener/inventory/pkg/utils/asynq"
	"github.com/gardener/inventory/pkg/utils/ptr"
)

// TaskCollectVirtualMachines is the name of the task for collecting Azure
// Virtual Machines.
const TaskCollectVirtualMachines = "az:task:collect-vms"

// CollectVirtualMachinesPayload is the payload used for collecting Azure
// Virtual Machines.
type CollectVirtualMachinesPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`
}

// NewCollectVirtualMachinesTask creates a new [asynq.Task] for collecting Azure
// Virtual Machines, without specifying a payload.
func NewCollectVirtualMachinesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectVirtualMachines, nil)
}

// HandleCollectVirtualMachinesTask is the handler, which collects Azure
// Virtual Machines.
func HandleCollectVirtualMachinesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known subscriptions.
	data := t.Payload()
	if data == nil {
		return enqueueCollectVirtualMachines(ctx)
	}

	var payload CollectVirtualMachinesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}

	return collectVirtualMachines(ctx, payload)
}

// enqueueCollectVirtualMachines enqueues tasks for collecting Azure Virtual
// Machines for all known Resource Groups.
func enqueueCollectVirtualMachines(ctx context.Context) error {
	resourceGroups, err := azureutils.GetResourceGroupsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	for _, rg := range resourceGroups {
		if !azureclients.VirtualMachinesClientset.Exists(rg.SubscriptionID) {
			logger.Warn(
				"Azure VM client not found",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
			)
			continue
		}

		payload := CollectVirtualMachinesPayload{
			SubscriptionID: rg.SubscriptionID,
			ResourceGroup:  rg.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure VMs",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
				"reason", err,
			)
			continue
		}
		task := asynq.NewTask(TaskCollectVirtualMachines, data)
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

// collectVirtualMachines collects the Azure Virtual Machines from the
// subscription and resource group specified in the payload.
func collectVirtualMachines(ctx context.Context, payload CollectVirtualMachinesPayload) error {
	client, ok := azureclients.VirtualMachinesClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure VMs",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
	)

	items := make([]models.VirtualMachine, 0)
	pager := client.Client.NewListPager(
		payload.ResourceGroup,
		&armcompute.VirtualMachinesClientListOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure VMs",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"reason", err,
			)
			return azureutils.MaybeSkipRetry(err)
		}

		for _, vm := range page.Value {
			vmName := ptr.Value(vm.Name, "")
			var provisioningState string
			var vmSize armcompute.VirtualMachineSizeTypes
			var timeCreated time.Time
			if vm.Properties != nil {
				provisioningState = ptr.Value(vm.Properties.ProvisioningState, "")
				vmSize = ptr.Value(vm.Properties.HardwareProfile.VMSize, armcompute.VirtualMachineSizeTypes(""))
				timeCreated = ptr.Value(vm.Properties.TimeCreated, time.Time{})
			}

			// For each VM we need to make a separate API call in
			// order to get the runtime status information, which
			// will give us information about the power state of the
			// VM. Also, OSName, OSVersion and other fields are
			// always empty when returned by the Azure API, and for
			// that reason we are simply not collecting them.
			//
			// See [1] and [2] for more details.
			//
			// [1]: https://github.com/Azure/azure-sdk-for-go/issues/23298
			// [2]: https://github.com/Azure/azure-sdk-for-go/issues/18565
			instanceView, err := client.Client.InstanceView(
				ctx,
				payload.ResourceGroup,
				vmName,
				&armcompute.VirtualMachinesClientInstanceViewOptions{},
			)
			if err != nil {
				logger.Error(
					"unable to get Azure VM instance view",
					"subscription_id", payload.SubscriptionID,
					"resource_group", payload.ResourceGroup,
					"vm", vmName,
					"reason", err,
				)
				continue
			}

			var vmAgentVersion string
			if instanceView.VMAgent != nil {
				vmAgentVersion = ptr.Value(instanceView.VMAgent.VMAgentVersion, "")
			}

			item := models.VirtualMachine{
				Name:              vmName,
				SubscriptionID:    payload.SubscriptionID,
				ResourceGroupName: payload.ResourceGroup,
				Location:          ptr.Value(vm.Location, ""),
				ProvisioningState: provisioningState,
				TimeCreated:       timeCreated,
				HyperVGeneration:  string(ptr.Value(instanceView.HyperVGeneration, "")),
				VMSize:            string(vmSize),
				PowerState:        azureutils.GetPowerState(instanceView.Statuses),
				VMAgentVersion:    vmAgentVersion,
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
		Set("vm_created_at = EXCLUDED.vm_created_at").
		Set("hyper_v_gen = EXCLUDED.hyper_v_gen").
		Set("vm_size = EXCLUDED.vm_size").
		Set("power_state = EXCLUDED.power_state").
		Set("vm_agent_version = EXCLUDED.vm_agent_version").
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

	logger.Info("populated azure vms", "count", count)

	return nil
}
