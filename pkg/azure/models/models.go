// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// Names for the various models provided by this package.
// These names are used for registering models with [registry.ModelRegistry]
const (
	SubscriptionModelName                  = "az:model:subscription"
	ResourceGroupModelName                 = "az:model:resource_group"
	VirtualMachineModelName                = "az:model:vm"
	NetworkInterfaceModelName              = "az:model:network_interface"
	PublicAddressModelName                 = "az:model:public_address"
	LoadBalancerModelName                  = "az:model:loadbalancer"
	VPCModelName                           = "az:model:vpc"
	SubnetModelName                        = "az:model:subnet"
	StorageAccountModelName                = "az:model:storage_account"
	BlobContainerModelName                 = "az:model:blob_container"
	UserModelName                          = "az:model:user"
	ResourceGroupToSubscriptionModelName   = "az:model:link_rg_to_subscription"
	VirtualMachineToResourceGroupModelName = "az:model:link_vm_to_rg"
	PublicAddressToResourceGroupModelName  = "az:model:link_public_address_to_rg"
	LoadBalancerToResourceGroupModelName   = "az:model:link_lb_to_rg"
	VPCToResourceGroupModelName            = "az:model:link_vpc_to_rg"
	SubnetToVPCModelName                   = "az:model:link_subnet_to_vpc"
	BlobContainerToResourceGroupModelName  = "az:model:link_blob_container_to_rg"
)

// models specifies the mapping between name and model type, which will be
// registered with [registry.ModelRegistry].
var models = map[string]any{
	SubscriptionModelName:     &Subscription{},
	ResourceGroupModelName:    &ResourceGroup{},
	VirtualMachineModelName:   &VirtualMachine{},
	NetworkInterfaceModelName: &NetworkInterface{},
	PublicAddressModelName:    &PublicAddress{},
	LoadBalancerModelName:     &LoadBalancer{},
	VPCModelName:              &VPC{},
	SubnetModelName:           &Subnet{},
	StorageAccountModelName:   &StorageAccount{},
	BlobContainerModelName:    &BlobContainer{},
	UserModelName:             &User{},

	// Link models
	ResourceGroupToSubscriptionModelName:   &ResourceGroupToSubscription{},
	VirtualMachineToResourceGroupModelName: &VirtualMachineToResourceGroup{},
	PublicAddressToResourceGroupModelName:  &PublicAddressToResourceGroup{},
	LoadBalancerToResourceGroupModelName:   &LoadBalancerToResourceGroup{},
	VPCToResourceGroupModelName:            &VPCToResourceGroup{},
	SubnetToVPCModelName:                   &SubnetToVPC{},
	BlobContainerToResourceGroupModelName:  &BlobContainerToResourceGroup{},
}

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

	ResourceGroupID uuid.UUID `bun:"rg_id,notnull,type:uuid,unique:l_az_rg_to_subscription_key"`
	SubscriptionID  uuid.UUID `bun:"sub_id,notnull,type:uuid,unique:l_az_rg_to_subscription_key"`
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
	GalleryImageID    string         `bun:"gallery_image_id,nullzero"`
	Subscription      *Subscription  `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup     *ResourceGroup `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
}

// VirtualMachineToResourceGroup represents a link table connecting the
// [VirtualMachine] with [ResourceGroup] models.
type VirtualMachineToResourceGroup struct {
	bun.BaseModel `bun:"table:l_az_vm_to_rg"`
	coremodels.Model

	ResourceGroupID uuid.UUID `bun:"rg_id,notnull,type:uuid,unique:l_az_vm_to_rg_key"`
	VMID            uuid.UUID `bun:"vm_id,notnull,type:uuid,unique:l_az_vm_to_rg_key"`
}

// NetworkInterface represents an Azure Network Interface.
type NetworkInterface struct {
	bun.BaseModel `bun:"table:az_network_interface"`
	coremodels.Model

	Name                 string          `bun:"name,notnull,unique:az_network_interface_key"`
	SubscriptionID       string          `bun:"subscription_id,notnull,unique:az_network_interface_key"`
	ResourceGroupName    string          `bun:"resource_group,notnull,unique:az_network_interface_key"`
	Location             string          `bun:"location,notnull"`
	ProvisioningState    string          `bun:"provisioning_state,notnull"`
	MacAddress           string          `bun:"mac_address,nullzero"`
	NICType              string          `bun:"nic_type,nullzero"`
	PrimaryNIC           bool            `bun:"primary_nic,notnull"`
	VMName               string          `bun:"vm_name,nullzero"`
	VPCName              string          `bun:"vpc_name,nullzero"`
	SubnetName           string          `bun:"subnet_name,nullzero"`
	PrivateIP            net.IP          `bun:"private_ip,nullzero,type:inet"`
	PrivateIPAllocation  string          `bun:"private_ip_allocation,nullzero"`
	PublicIPName         string          `bun:"public_ip_name,nullzero"`
	NetworkSecurityGroup string          `bun:"network_security_group,nullzero"`
	IPForwardingEnabled  bool            `bun:"ip_forwarding_enabled,notnull"`
	Subscription         *Subscription   `bun:"rel:has-one,join:subscription_id=subscription_id"`
	ResourceGroup        *ResourceGroup  `bun:"rel:has-one,join:resource_group=name,join:subscription_id=subscription_id"`
	VirtualMachine       *VirtualMachine `bun:"rel:has-one,join:vm_name=name,join:subscription_id=subscription_id,join:resource_group=resource_group"`
	VPC                  *VPC            `bun:"rel:has-one,join:vpc_name=name,join:subscription_id=subscription_id,join:resource_group=resource_group"`
	Subnet               *Subnet         `bun:"rel:has-one,join:subnet_name=name,join:vpc_name=vpc_name,join:subscription_id=subscription_id,join:resource_group=resource_group"`
	PublicAddress        *PublicAddress  `bun:"rel:has-one,join:public_ip_name=name,join:subscription_id=subscription_id,join:resource_group=resource_group"`
}

// NetworkInterfaceToVM represents a link table connecting the
// [NetworkInterface] with [VirtualMachine] models.
type NetworkInterfaceToVM struct {
	bun.BaseModel `bun:"table:l_az_nic_to_vm"`
	coremodels.Model

	NetworkInterfaceID uuid.UUID `bun:"nic_id,notnull,type:uuid,unique:l_az_nic_to_vm_key"`
	VMID               uuid.UUID `bun:"vm_id,notnull,type:uuid,unique:l_az_nic_to_vm_key"`
}

// NetworkInterfaceToPublicAddress represents a link table connecting the
// [NetworkInterface] with [PublicAddress] models.
type NetworkInterfaceToPublicAddress struct {
	bun.BaseModel `bun:"table:l_az_nic_to_pub_addr"`
	coremodels.Model

	NetworkInterfaceID uuid.UUID `bun:"nic_id,notnull,type:uuid,unique:l_az_nic_to_pub_addr_key"`
	PublicAddressID    uuid.UUID `bun:"pa_id,notnull,type:uuid,unique:l_az_nic_to_pub_addr_key"`
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

	ResourceGroupID uuid.UUID `bun:"rg_id,notnull,type:uuid,unique:l_az_pub_addr_to_rg_key"`
	PublicAddressID uuid.UUID `bun:"pa_id,notnull,type:uuid,unique:l_az_pub_addr_to_rg_key"`
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

	ResourceGroupID uuid.UUID `bun:"rg_id,notnull,type:uuid,unique:l_az_lb_to_rg_key"`
	LoadBalancerID  uuid.UUID `bun:"lb_id,notnull,type:uuid,unique:l_az_lb_to_rg_key"`
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
	VPC               *VPC           `bun:"rel:has-one,join:vpc_name=name,join:subscription_id=subscription_id,join:resource_group=resource_group"`
}

// VPCToResourceGroup represents a link table connecting the
// [VPC] with [ResourceGroup] models.
type VPCToResourceGroup struct {
	bun.BaseModel `bun:"table:l_az_vpc_to_rg"`
	coremodels.Model

	VPCID           uuid.UUID `bun:"vpc_id,notnull,type:uuid,unique:l_az_vpc_to_rg_key"`
	ResourceGroupID uuid.UUID `bun:"rg_id,notnull,type:uuid,unique:l_az_vpc_to_rg_key"`
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

// SubnetToVPC represents a link table connecting the
// [Subnet] with [VPC] models.
type SubnetToVPC struct {
	bun.BaseModel `bun:"table:l_az_subnet_to_vpc"`
	coremodels.Model

	SubnetID uuid.UUID `bun:"subnet_id,notnull,type:uuid,unique:l_az_subnet_to_vpc_key"`
	VPCID    uuid.UUID `bun:"vpc_id,notnull,type:uuid,unique:l_az_subnet_to_vpc_key"`
}

// BlobContainerToResourceGroup represents a link table connecting the
// [BlobContainer] with [ResourceGroup] models.
type BlobContainerToResourceGroup struct {
	bun.BaseModel `bun:"table:l_az_blob_container_to_rg"`
	coremodels.Model

	BlobContainerID uuid.UUID `bun:"blob_container_id,notnull,type:uuid,unique:l_az_blob_container_to_rg_key"`
	ResourceGroupID uuid.UUID `bun:"rg_id,notnull,type:uuid,unique:l_az_blob_container_to_rg_key"`
}

// User represents a Microsoft Entra user account.
type User struct {
	bun.BaseModel `bun:"table:az_user"`
	coremodels.Model

	UserID   string `bun:"user_id,notnull,unique:az_user_key"`
	TenantID string `bun:"tenant_id,notnull,unique:az_user_key"`
	Mail     string `bun:"mail,notnull"`
}

// init registers the models with the [registry.ModelRegistry].
func init() {
	for k, v := range models {
		registry.ModelRegistry.MustRegister(k, v)
	}
}
