// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"time"

	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// Subscription represents an Azure Subscription
type Subscription struct {
	bun.BaseModel `bun:"table:az_subscription"`
	coremodels.Model

	SubscriptionID string           `bun:"subscription_id,notnull,unique"`
	Name           string           `bun:"name,nullzero"`
	State          string           `bun:"state,nullzero"`
	ResourceGroups []*ResourceGroup `bun:"rel:has-many,join:subscription_id=subscription_id"`
}

// ResourceGroup represents an Azure Resource Group
type ResourceGroup struct {
	bun.BaseModel `bun:"table:az_resource_group"`
	coremodels.Model

	Name           string        `bun:"name,notnull,unique:az_resource_group_key"`
	SubscriptionID string        `bun:"subscription_id,notnull,unique:az_resource_group_key"`
	Location       string        `bun:"location,notnull"`
	Subscription   *Subscription `bun:"rel:has-one,join:subscription_id=subscription_id"`
}

// ResourceGroupToSubscription represents a link table connecting the
// [Subscription] with [ResourceGroup] models.
type ResourceGroupToSubscription struct {
	bun.BaseModel `bun:"table:l_az_rg_to_subscription"`
	coremodels.Model

	ResourceGroupID uint64 `bun:"rg_id,notnull,unique:l_az_rg_to_subscription_key"`
	SubscriptionID  uint64 `bun:"sub_id,notnull,unique:l_az_rg_to_subscription_key"`
}

// VirtualMachine represents an Azure Virtual Machine.
type VirtualMachine struct {
	bun.BaseModel `bun:"table:az_vm"`
	coremodels.Model

	Name              string         `bun:"name,notnull,unique:az_vm_key"`
	SubscriptionID    string         `bun:"subscription_id,notnull,unique:az_vm_key"`
	ResourceGroupName string         `bun:"resource_group,notnull,unique:az_vm_key"`
	Location          string         `bun:"location,notnull"`
	ProvisioningState string         `bun:"provisioning_state,notnull"`
	TimeCreated       time.Time      `bun:"vm_created_at,nullzero"`
	VMSize            string         `bun:"vm_size,nullzero"`
	PowerState        string         `bun:"power_state,nullzero"`
	HyperVGeneration  string         `bun:"hyper_v_gen,nullzero"`
	VMAgentVersion    string         `bun:"vm_agent_version,nullzero"`
	Subscription      *Subscription  `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup     *ResourceGroup `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("az:model:subscription", &Subscription{})
	registry.ModelRegistry.MustRegister("az:model:resource_group", &ResourceGroup{})
	registry.ModelRegistry.MustRegister("az:model:vm", &VirtualMachine{})

	// Link tables
	registry.ModelRegistry.MustRegister("az:model:link_rg_to_subscription", &ResourceGroupToSubscription{})
}
