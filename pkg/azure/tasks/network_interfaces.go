// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"net"

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

// TaskCollectNetworkInterfaces is the name of the task for collecting Azure
// Network Interfaces.
const TaskCollectNetworkInterfaces = "az:task:collect-network-interfaces"

// CollectNetworkInterfacesPayload is the payload used for collecting Azure
// Network Interfaces.
type CollectNetworkInterfacesPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`
}

// NewCollectNetworkInterfacesTask creates a new [asynq.Task] for collecting Azure
// Network Interfaces, without specifying a payload.
func NewCollectNetworkInterfacesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectNetworkInterfaces, nil)
}

// HandleCollectNetworkInterfacesTask is the handler, which collects Azure
// Network Interfaces.
func HandleCollectNetworkInterfacesTask(ctx context.Context, t *asynq.Task) error {
	data := t.Payload()
	if data == nil {
		return enqueueCollectNetworkInterfaces(ctx)
	}

	var payload CollectNetworkInterfacesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}

	return collectNetworkInterfaces(ctx, payload)
}

// enqueueCollectNetworkInterfaces enqueues tasks for collecting Azure Network
// Interfaces for known Resource Groups.
func enqueueCollectNetworkInterfaces(ctx context.Context) error {
	resourceGroups, err := azureutils.GetResourceGroupsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	queue := asynqutils.GetQueueName(ctx)
	for _, rg := range resourceGroups {
		if !azureclients.NetworkInterfacesClientset.Exists(rg.SubscriptionID) {
			logger.Warn(
				"azure network interfaces client not found",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
			)

			continue
		}

		payload := CollectNetworkInterfacesPayload{
			SubscriptionID: rg.SubscriptionID,
			ResourceGroup:  rg.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for azure network interfaces",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
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

// collectNetworkInterfaces collects the Azure Network Interfaces from the
// subscription and resource group specified in the payload.
func collectNetworkInterfaces(ctx context.Context, payload CollectNetworkInterfacesPayload) error {
	client, ok := azureclients.NetworkInterfacesClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting azure network interfaces",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
	)

	var count int64
	defer func() {
		metric := prometheus.MustNewConstMetric(
			networkInterfacesDesc,
			prometheus.GaugeValue,
			float64(count),
			payload.SubscriptionID,
			payload.ResourceGroup,
		)
		key := metrics.Key(TaskCollectNetworkInterfaces, payload.SubscriptionID, payload.ResourceGroup)
		metrics.DefaultCollector.AddMetric(key, metric)
	}()

	items := make([]models.NetworkInterface, 0)
	pager := client.Client.NewListPager(
		payload.ResourceGroup,
		&armnetwork.InterfacesClientListOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get azure network interfaces",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"reason", err,
			)

			return azureutils.MaybeSkipRetry(err)
		}

		for _, nic := range page.Value {
			if nic == nil {
				logger.Warn(
					"azure Network Interfaces client not found",
					"subscription_id", payload.SubscriptionID,
					"resource_group", payload.ResourceGroup,
				)

				continue
			}

			item := extractNIC(ctx, *nic, payload.SubscriptionID, payload.ResourceGroup)

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
		Set("mac_address = EXCLUDED.mac_address").
		Set("nic_type = EXCLUDED.nic_type").
		Set("primary_nic = EXCLUDED.primary_nic").
		Set("vm_name = EXCLUDED.vm_name").
		Set("vpc_name = EXCLUDED.vpc_name").
		Set("subnet_name = EXCLUDED.subnet_name").
		Set("private_ip = EXCLUDED.private_ip").
		Set("private_ip_allocation = EXCLUDED.private_ip_allocation").
		Set("public_ip_name = EXCLUDED.public_ip_name").
		Set("network_security_group = EXCLUDED.network_security_group").
		Set("ip_forwarding_enabled = EXCLUDED.ip_forwarding_enabled").
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

	logger.Info("populated azure network interfaces", "count", count)

	return nil
}

func extractNIC(ctx context.Context, nic armnetwork.Interface, subscriptionID string, resourceGroup string) models.NetworkInterface {
	logger := asynqutils.GetLogger(ctx)

	var provisioningState armnetwork.ProvisioningState
	var macAddress, nicType string
	var primaryNIC bool
	var vmName, vpcName, subnetName, publicIPName, nsgName string
	var privateIP net.IP
	var privateIPAllocation string
	var ipForwardingEnabled bool

	name := ptr.Value(nic.Name, "")
	if name == "" {
		logger.Error(
			"failed getting azure network interfaces",
			"subscription_id", subscriptionID,
			"resource_group", resourceGroup,
			"nic id", ptr.Value(nic.ID, ""),
			"reason", "missing name in resource",
		)
	}

	if nic.Properties != nil {
		provisioningState = ptr.Value(nic.Properties.ProvisioningState, "")
		macAddress = ptr.Value(nic.Properties.MacAddress, "")
		nicType = string(ptr.Value(nic.Properties.NicType, ""))
		primaryNIC = ptr.Value(nic.Properties.Primary, false)
		ipForwardingEnabled = ptr.Value(nic.Properties.EnableIPForwarding, false)

		if nic.Properties.VirtualMachine != nil && nic.Properties.VirtualMachine.ID != nil {
			vmName = azureutils.ExtractResourceNameFromID(ptr.Value(nic.Properties.VirtualMachine.ID, ""))
		}

		if nic.Properties.NetworkSecurityGroup != nil && nic.Properties.NetworkSecurityGroup.ID != nil {
			nsgName = azureutils.ExtractResourceNameFromID(ptr.Value(nic.Properties.NetworkSecurityGroup.ID, ""))
		}

		if len(nic.Properties.IPConfigurations) > 0 {
			var primaryIPConfig *armnetwork.InterfaceIPConfiguration
			for _, ipConfig := range nic.Properties.IPConfigurations {
				if ipConfig != nil && ipConfig.Properties != nil && ptr.Value(ipConfig.Properties.Primary, false) {
					primaryIPConfig = ipConfig

					break
				}
			}
			if primaryIPConfig == nil {
				primaryIPConfig = nic.Properties.IPConfigurations[0]
			}

			if primaryIPConfig != nil && primaryIPConfig.Properties != nil {
				privateIP = net.ParseIP(ptr.Value(primaryIPConfig.Properties.PrivateIPAddress, ""))
				privateIPAllocation = string(ptr.Value(primaryIPConfig.Properties.PrivateIPAllocationMethod, ""))

				if primaryIPConfig.Properties.Subnet != nil && primaryIPConfig.Properties.Subnet.ID != nil {
					subnetID := ptr.Value(primaryIPConfig.Properties.Subnet.ID, "")
					if subnetID != "" {
						subnetName = azureutils.ExtractResourceNameFromID(subnetID)
						vpcName = azureutils.ExtractParentResourceNameFromID(subnetID)
					}
				}

				if primaryIPConfig.Properties.PublicIPAddress != nil && primaryIPConfig.Properties.PublicIPAddress.ID != nil {
					publicIPName = azureutils.ExtractResourceNameFromID(ptr.Value(primaryIPConfig.Properties.PublicIPAddress.ID, ""))
				}
			}
		}
	}

	item := models.NetworkInterface{
		Name:                 name,
		SubscriptionID:       subscriptionID,
		ResourceGroupName:    resourceGroup,
		Location:             ptr.Value(nic.Location, ""),
		ProvisioningState:    string(provisioningState),
		MacAddress:           macAddress,
		NICType:              nicType,
		PrimaryNIC:           primaryNIC,
		VMName:               vmName,
		VPCName:              vpcName,
		SubnetName:           subnetName,
		PrivateIP:            privateIP,
		PrivateIPAllocation:  privateIPAllocation,
		PublicIPName:         publicIPName,
		NetworkSecurityGroup: nsgName,
		IPForwardingEnabled:  ipForwardingEnabled,
	}

	return item
}
