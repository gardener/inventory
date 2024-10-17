// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"net"
	"time"

	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// Subscription represents an Azure Subscription
type Subscription struct {
	bun.BaseModel `bun:"table:az_subscription"`
	coremodels.Model

	SubscriptionID  string            `bun:"subscription_id,notnull,unique"`
	Name            string            `bun:"name,nullzero"`
	State           string            `bun:"state,nullzero"`
	ResourceGroups  []*ResourceGroup  `bun:"rel:has-many,join:subscription_id=subscription_id"`
	VirtualMachines []*VirtualMachine `bun:"rel:has-many,join:subscription_id=subscription_id"`
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

// VirtualMachineToResourceGroup represents a link table connecting the
// [VirtualMachine] with [ResourceGroup] models.
type VirtualMachineToResourceGroup struct {
	bun.BaseModel `bun:"table:l_az_vm_to_rg"`
	coremodels.Model

	ResourceGroupID uint64 `bun:"rg_id,notnull,unique:l_az_vm_to_rg_key"`
	VMID            uint64 `bun:"vm_id,notnull,unique:l_az_vm_to_rg_key"`
}

// PublicAddress represents an Azure Public IP Address.
type PublicAddress struct {
	bun.BaseModel `bun:"table:az_public_address"`
	coremodels.Model

	Name              string         `bun:"name,notnull,unique:az_public_address_key"`
	SubscriptionID    string         `bun:"subscription_id,notnull,unique:az_public_address_key"`
	ResourceGroupName string         `bun:"resource_group,notnull,unique:az_public_address_key"`
	Location          string         `bun:"location,notnull"`
	ProvisioningState string         `bun:"provisioning_state,notnull"`
	SKUName           string         `bun:"sku_name,notnull"`
	SKUTier           string         `bun:"sku_tier,notnull"`
	DDoSProctection   string         `bun:"ddos_protection,nullzero"`
	FQDN              string         `bun:"fqdn,nullzero"`
	ReverseFQDN       string         `bun:"reverse_fqdn,nullzero"`
	NATGateway        string         `bun:"nat_gateway,nullzero"`
	IPAddress         net.IP         `bun:"ip_address,nullzero,type:inet"`
	IPVersion         string         `bun:"ip_version,nullzero"`
	Subscription      *Subscription  `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup     *ResourceGroup `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
}

// PublicAddressToResourceGroup represents a link table connecting the
// [PublicAddress] with [ResourceGroup] models.
type PublicAddressToResourceGroup struct {
	bun.BaseModel `bun:"table:l_az_pub_addr_to_rg"`
	coremodels.Model

	ResourceGroupID uint64 `bun:"rg_id,notnull,unique:l_az_pub_addr_to_rg_key"`
	PublicAddressID uint64 `bun:"pa_id,notnull,unique:l_az_pub_addr_to_rg_key"`
}

// LoadBalancer represents an Azure Load Balancer.
type LoadBalancer struct {
	bun.BaseModel `bun:"table:az_lb"`
	coremodels.Model

	Name              string         `bun:"name,notnull,unique:az_lb_key"`
	SubscriptionID    string         `bun:"subscription_id,notnull,unique:az_lb_key"`
	ResourceGroupName string         `bun:"resource_group,notnull,unique:az_lb_key"`
	Location          string         `bun:"location,notnull"`
	ProvisioningState string         `bun:"provisioning_state,notnull"`
	SKUName           string         `bun:"sku_name,notnull"`
	SKUTier           string         `bun:"sku_tier,notnull"`
	Subscription      *Subscription  `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup     *ResourceGroup `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
}

// LoadBalancerToResourceGroup represents a link table connecting the
// [LoadBalancer] with [ResourceGroup] models.
type LoadBalancerToResourceGroup struct {
	bun.BaseModel `bun:"table:l_az_lb_to_rg"`
	coremodels.Model

	ResourceGroupID uint64 `bun:"rg_id,notnull,unique:l_az_lb_to_rg_key"`
	LoadBalancerID  uint64 `bun:"lb_id,notnull,unique:l_az_lb_to_rg_key"`
}

// VPC represents an Azure VPC.
type VPC struct {
	bun.BaseModel `bun:"table:az_vpc"`
	coremodels.Model

	Name                string         `bun:"name,notnull,unique:az_vpc_key"`
	SubscriptionID      string         `bun:"subscription_id,notnull,unique:az_vpc_key"`
	ResourceGroupName   string         `bun:"resource_group,notnull,unique:az_vpc_key"`
	Location            string         `bun:"location,notnull"`
	ProvisioningState   string         `bun:"provisioning_state,notnull"`
	EncryptionEnabled   bool           `bun:"encryption_enabled,notnull"`
	VMProtectionEnabled bool           `bun:"vm_protection_enabled,notnull"`
	Subscription        *Subscription  `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup       *ResourceGroup `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
}

// Subnet represents an Azure Subnet.
type Subnet struct {
	bun.BaseModel `bun:"table:az_subnet"`
	coremodels.Model

	Name              string         `bun:"name,notnull,unique:az_subnet_key"`
	SubscriptionID    string         `bun:"subscription_id,notnull,unique:az_subnet_key"`
	ResourceGroupName string         `bun:"resource_group,notnull,unique:az_subnet_key"`
	VPCName           string         `bun:"vpc_name,notnull,unique:az_subnet_key"`
	Type              string         `bun:"type,notnull"`
	ProvisioningState string         `bun:"provisioning_state,notnull"`
	AddressPrefix     string         `bun:"address_prefix,notnull"`
	SecurityGroup     string         `bun:"security_group,notnull"`
	Purpose           string         `bun:"purpose,notnull"`
	Subscription      *Subscription  `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup     *ResourceGroup `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
	VPC               *VPC           `bun:"rel:has-one,join:vpc_name=name,subscription_id=subscription_id,resource_group=resource_group"`
}

// VPCToResourceGroup represents a link table connecting the
// [VPC] with [ResourceGroup] models.
type VPCToResourceGroup struct {
	bun.BaseModel `bun:"table:l_az_vpc_to_rg"`
	coremodels.Model

	VPCID           uint64 `bun:"vpc_id,notnull,unique:l_az_vpc_to_rg_key"`
	ResourceGroupID uint64 `bun:"rg_id,notnull,unique:l_az_vpc_to_rg_key"`
}

// StorageAccount represents an Azure Storage Account.
type StorageAccount struct {
	bun.BaseModel `bun:"table:az_storage_account"`
	coremodels.Model

	Name              string         `bun:"name,notnull,unique:az_storage_account_key"`
	SubscriptionID    string         `bun:"subscription_id,notnull,unique:az_storage_account_key"`
	ResourceGroupName string         `bun:"resource_group,notnull,unique:az_storage_account_key"`
	Location          string         `bun:"location,notnull"`
	ProvisioningState string         `bun:"provisioning_state,notnull"`
	Kind              string         `bun:"kind,notnull"`
	SKUName           string         `bun:"sku_name,notnull"`
	SKUTier           string         `bun:"sku_tier,notnull"`
	CreationTime      time.Time      `bun:"creation_time,nullzero"`
	Subscription      *Subscription  `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup     *ResourceGroup `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
}

// BlobContainer represents an Azure Blob container.
type BlobContainer struct {
	bun.BaseModel `bun:"table:az_blob_container"`
	coremodels.Model

	Name               string          `bun:"name,notnull,unique:az_blob_container_key"`
	SubscriptionID     string          `bun:"subscription_id,notnull,unique:az_blob_container_key"`
	ResourceGroupName  string          `bun:"resource_group,notnull,unique:az_blob_container_key"`
	StorageAccountName string          `bun:"storage_account,notnull,unique:az_blob_container_key"`
	PublicAccess       string          `bun:"public_access,notnull"`
	Deleted            bool            `bun:"deleted,notnull"`
	LastModifiedTime   time.Time       `bun:"last_modified_time,nullzero"`
	Subscription       *Subscription   `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup      *ResourceGroup  `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
	StorageAccount     *StorageAccount `bun:"rel:has-one,join:storage_account=name,join:resource_group=resource_group,join:subscription_id=subscription_id"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("az:model:subscription", &Subscription{})
	registry.ModelRegistry.MustRegister("az:model:resource_group", &ResourceGroup{})
	registry.ModelRegistry.MustRegister("az:model:vm", &VirtualMachine{})
	registry.ModelRegistry.MustRegister("az:model:public_address", &PublicAddress{})
	registry.ModelRegistry.MustRegister("az:model:loadbalancer", &LoadBalancer{})
	registry.ModelRegistry.MustRegister("az:model:vpc", &VPC{})
	registry.ModelRegistry.MustRegister("az:model:subnet", &Subnet{})
	registry.ModelRegistry.MustRegister("az:model:storage_account", &StorageAccount{})
	registry.ModelRegistry.MustRegister("az:model:blob_container", &BlobContainer{})

	// Link tables
	registry.ModelRegistry.MustRegister("az:model:link_rg_to_subscription", &ResourceGroupToSubscription{})
	registry.ModelRegistry.MustRegister("az:model:link_vm_to_rg", &VirtualMachineToResourceGroup{})
	registry.ModelRegistry.MustRegister("az:model:link_public_address_to_rg", &PublicAddressToResourceGroup{})
	registry.ModelRegistry.MustRegister("az:model:link_lb_to_rg", &LoadBalancerToResourceGroup{})
	registry.ModelRegistry.MustRegister("az:model:link_vpc_to_rg", &VPCToResourceGroup{})
}
