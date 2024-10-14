// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"context"
	"encoding/json"
	"net"

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

// TaskCollectPublicAddresses is the name of the task for collecting Azure
// Public IP Addresses.
const TaskCollectPublicAddresses = "az:task:collect-public-addresses"

// CollectPublicAddressesPayload is the payload used for collecting Azure
// Public IP Addresses.
type CollectPublicAddressesPayload struct {
	// SubscriptionID specifies the Azure Subscription ID from which to
	// collect.
	SubscriptionID string `json:"subscription_id" yaml:"subscription_id"`

	// ResourceGroup specifies from which resource group to collect.
	ResourceGroup string `json:"resource_group" yaml:"resource_group"`
}

// NewCollectPublicAddressesTask creates a new [asynq.Task] for collecting Azure
// Public IP Addresses, without specifying a payload.
func NewCollectPublicAddressesTask() *asynq.Task {
	return asynq.NewTask(TaskCollectPublicAddresses, nil)
}

// HandleCollectPublicAddressesTask is the handler, which collects Azure
// Public IP Addresses.
func HandleCollectPublicAddressesTask(ctx context.Context, t *asynq.Task) error {
	// If we were called without a payload, then we enqueue collection from
	// all known resource groups.
	data := t.Payload()
	if data == nil {
		return enqueueCollectPublicAddresses(ctx)
	}

	var payload CollectPublicAddressesPayload
	if err := asynqutils.Unmarshal(data, &payload); err != nil {
		return asynqutils.SkipRetry(err)
	}

	if payload.SubscriptionID == "" {
		return asynqutils.SkipRetry(ErrNoSubscriptionID)
	}
	if payload.ResourceGroup == "" {
		return asynqutils.SkipRetry(ErrNoResourceGroup)
	}

	return collectPublicAddresses(ctx, payload)
}

// enqueuePublicAddresses enqueues tasks for collecting Azure Public IP
// Addresses for known Resource Groups.
func enqueueCollectPublicAddresses(ctx context.Context) error {
	resourceGroups, err := azureutils.GetResourceGroupsFromDB(ctx)
	if err != nil {
		return err
	}

	// Enqueue task for each resource group
	logger := asynqutils.GetLogger(ctx)
	for _, rg := range resourceGroups {
		if !azureclients.PublicIPAddressesClientset.Exists(rg.SubscriptionID) {
			logger.Warn(
				"Azure Public Addresses client not found",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
			)
			continue
		}

		payload := CollectPublicAddressesPayload{
			SubscriptionID: rg.SubscriptionID,
			ResourceGroup:  rg.Name,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			logger.Error(
				"failed to marshal payload for Azure Public Addresses",
				"subscription_id", rg.SubscriptionID,
				"resource_group", rg.Name,
				"reason", err,
			)
			continue
		}
		task := asynq.NewTask(TaskCollectPublicAddresses, data)
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

// collectPublicAddresses collects the Azure Public IP Addresses from the
// subscription and resource group specified in the payload.
func collectPublicAddresses(ctx context.Context, payload CollectPublicAddressesPayload) error {
	client, ok := azureclients.PublicIPAddressesClientset.Get(payload.SubscriptionID)
	if !ok {
		return asynqutils.SkipRetry(ClientNotFound(payload.SubscriptionID))
	}

	logger := asynqutils.GetLogger(ctx)
	logger.Info(
		"collecting Azure Public Addresses",
		"subscription_id", payload.SubscriptionID,
		"resource_group", payload.ResourceGroup,
	)

	items := make([]models.PublicAddress, 0)
	pager := client.Client.NewListPager(
		payload.ResourceGroup,
		&armnetwork.PublicIPAddressesClientListOptions{},
	)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.Error(
				"failed to get Azure Public Addresses",
				"subscription_id", payload.SubscriptionID,
				"resource_group", payload.ResourceGroup,
				"reason", err,
			)
			return err
		}

		for _, addr := range page.Value {
			var provisioningState armnetwork.ProvisioningState
			var ddosProtection armnetwork.DdosSettingsProtectionMode
			var ipVersion armnetwork.IPVersion
			var ipAddr net.IP
			var fqdn, reverseFQDN, natGatewayName string
			if addr.Properties != nil {
				provisioningState = ptr.Value(addr.Properties.ProvisioningState, armnetwork.ProvisioningState(""))
				ipAddr = net.ParseIP(ptr.Value(addr.Properties.IPAddress, ""))
				ipVersion = ptr.Value(addr.Properties.PublicIPAddressVersion, armnetwork.IPVersion(""))
				if addr.Properties.DdosSettings != nil {
					ddosProtection = ptr.Value(addr.Properties.DdosSettings.ProtectionMode, armnetwork.DdosSettingsProtectionMode(""))
				}
				if addr.Properties.DNSSettings != nil {
					fqdn = ptr.Value(addr.Properties.DNSSettings.Fqdn, "")
					reverseFQDN = ptr.Value(addr.Properties.DNSSettings.ReverseFqdn, "")
				}
				if addr.Properties.NatGateway != nil {
					natGatewayName = ptr.Value(addr.Properties.NatGateway.Name, "")
				}
			}
			var skuName armnetwork.PublicIPAddressSKUName
			var skuTier armnetwork.PublicIPAddressSKUTier
			if addr.SKU != nil {
				skuName = ptr.Value(addr.SKU.Name, armnetwork.PublicIPAddressSKUName(""))
				skuTier = ptr.Value(addr.SKU.Tier, armnetwork.PublicIPAddressSKUTier(""))
			}

			item := models.PublicAddress{
				Name:              ptr.Value(addr.Name, ""),
				SubscriptionID:    payload.SubscriptionID,
				ResourceGroupName: payload.ResourceGroup,
				Location:          ptr.Value(addr.Location, ""),
				ProvisioningState: string(provisioningState),
				DDoSProctection:   string(ddosProtection),
				FQDN:              fqdn,
				ReverseFQDN:       reverseFQDN,
				NATGateway:        natGatewayName,
				IPAddress:         ipAddr,
				IPVersion:         string(ipVersion),
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
		Set("ddos_protection = EXCLUDED.ddos_protection").
		Set("fqdn = EXCLUDED.fqdn").
		Set("reverse_fqdn = EXCLUDED.reverse_fqdn").
		Set("nat_gateway = EXCLUDED.nat_gateway").
		Set("ip_address = EXCLUDED.ip_address").
		Set("ip_version = EXCLUDED.ip_version").
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

	logger.Info("populated azure public ip address", "count", count)

	return nil
}
