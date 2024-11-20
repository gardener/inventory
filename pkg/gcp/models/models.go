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

// Project represents a GCP Project.
type Project struct {
	bun.BaseModel `bun:"table:gcp_project"`
	coremodels.Model

	// Name is the globally unique id of the project represented as
	// "projects/<uint64>" value
	Name string `bun:"name,notnull,unique"`
	// ProjectID is the user-defined globally unique project id.
	ProjectID string `bun:"project_id,notnull,unique"`

	Parent            string      `bun:"parent,notnull"`
	State             string      `bun:"state,notnull"`
	DisplayName       string      `bun:"display_name,notnull"`
	ProjectCreateTime time.Time   `bun:"project_create_time,nullzero"`
	ProjectUpdateTime time.Time   `bun:"project_update_time,nullzero"`
	ProjectDeleteTime time.Time   `bun:"project_delete_time,nullzero"`
	Etag              string      `bun:"etag,notnull"`
	Instances         []*Instance `bun:"rel:has-many,join:project_id=project_id"`
	VPCs              []*VPC      `bun:"rel:has-many,join:project_id=project_id"`
}

// Instance represents a GCP Instance.
type Instance struct {
	bun.BaseModel `bun:"table:gcp_instance"`
	coremodels.Model

	Name                 string   `bun:"name,notnull"`
	Hostname             string   `bun:"hostname,notnull"`
	InstanceID           uint64   `bun:"instance_id,notnull,unique:gcp_instance_key"`
	ProjectID            string   `bun:"project_id,notnull,unique:gcp_instance_key"`
	Zone                 string   `bun:"zone,notnull"`
	Region               string   `bun:"region,notnull"`
	CanIPForward         bool     `bun:"can_ip_forward,notnull"`
	CPUPlatform          string   `bun:"cpu_platform,notnull"`
	CreationTimestamp    string   `bun:"creation_timestamp,nullzero"`
	Description          string   `bun:"description,notnull"`
	LastStartTimestamp   string   `bun:"last_start_timestamp,nullzero"`
	LastStopTimestamp    string   `bun:"last_stop_timestamp,nullzero"`
	LastSuspendTimestamp string   `bun:"last_suspend_timestamp,nullzero"`
	MachineType          string   `bun:"machine_type,notnull"`
	MinCPUPlatform       string   `bun:"min_cpu_platform,notnull"`
	SelfLink             string   `bun:"self_link,notnull"`
	SourceMachineImage   string   `bun:"source_machine_image,notnull"`
	Status               string   `bun:"status,notnull"`
	StatusMessage        string   `bun:"status_message,notnull"`
	Project              *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// NetworkInterface represents a NIC attached to an [Instance].
type NetworkInterface struct {
	bun.BaseModel `bun:"table:gcp_nic"`
	coremodels.Model

	Name           string    `bun:"name,notnull,unique:gcp_nic_key"`
	ProjectID      string    `bun:"project_id,notnull,unique:gcp_nic_key"`
	InstanceID     uint64    `bun:"instance_id,notnull,unique:gcp_nic_key"`
	Network        string    `bun:"network,notnull"`
	Subnetwork     string    `bun:"subnetwork,notnull"`
	IPv4           net.IP    `bun:"ipv4,nullzero,type:inet"`
	IPv6           net.IP    `bun:"ipv6,nullzero,type:inet"`
	IPv6AccessType string    `bun:"ipv6_access_type,notnull"`
	NICType        string    `bun:"nic_type,notnull"`
	StackType      string    `bun:"stack_type,notnull"`
	Instance       *Instance `bun:"rel:has-one,join:project_id=project_id,join:instance_id=instance_id"`
}

// InstanceToNetworkInterface represents a link table connecting the
// [NetworkInterface] with [Instance] models.
type InstanceToNetworkInterface struct {
	bun.BaseModel `bun:"table:l_gcp_instance_to_nic"`
	coremodels.Model

	InstanceID         uint64 `bun:"instance_id,notnull,unique:l_gcp_instance_to_nic_key"`
	NetworkInterfaceID uint64 `bun:"nic_id,notnull,unique:l_gcp_instance_to_nic_key"`
}

// InstanceToProject represents a link table connecting the [Project] with
// [Instance] models.
type InstanceToProject struct {
	bun.BaseModel `bun:"table:l_gcp_instance_to_project"`
	coremodels.Model

	ProjectID  uint64 `bun:"project_id,notnull,unique:l_gcp_instance_to_project_key"`
	InstanceID uint64 `bun:"instance_id,notnull,unique:l_gcp_instance_to_project_key"`
}

// VPC represents a GCP VPC
type VPC struct {
	bun.BaseModel `bun:"table:gcp_vpc"`
	coremodels.Model

	VPCID             uint64   `bun:"vpc_id,notnull,unique:gcp_vpc_key"`
	ProjectID         string   `bun:"project_id,notnull,unique:gcp_vpc_key"`
	Name              string   `bun:"name,notnull"`
	CreationTimestamp string   `bun:"creation_timestamp,nullzero"`
	Description       string   `bun:"description,notnull"`
	GatewayIPv4       string   `bun:"gateway_ipv4,notnull"`
	FirewallPolicy    string   `bun:"firewall_policy,notnull"`
	MTU               int32    `bun:"mtu,notnull"`
	Project           *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// VPCToProject represents a link table connecting the [Project] with
// [VPC] models.
type VPCToProject struct {
	bun.BaseModel `bun:"table:l_gcp_vpc_to_project"`
	coremodels.Model

	ProjectID uint64 `bun:"project_id,notnull,unique:l_gcp_vpc_to_project_key"`
	VPCID     uint64 `bun:"vpc_id,notnull,unique:l_gcp_vpc_to_project_key"`
}

// Address represents a GCP static IP address resource. The Address model
// represents both - a global (external and internal) and regional (external and
// internal) IP address.
type Address struct {
	bun.BaseModel `bun:"table:gcp_address"`
	coremodels.Model

	Address           net.IP   `bun:"address,notnull,type:inet"`
	AddressType       string   `bun:"address_type,notnull"`
	IsGlobal          bool     `bun:"is_global,notnull"`
	CreationTimestamp string   `bun:"creation_timestamp,nullzero"`
	Description       string   `bun:"description,notnull"`
	AddressID         uint64   `bun:"address_id,notnull,unique:gcp_address_key"`
	ProjectID         string   `bun:"project_id,notnull,unique:gcp_address_key"`
	Region            string   `bun:"region,notnull"`
	IPVersion         string   `bun:"ip_version,notnull"`
	IPv6EndpointType  string   `bun:"ipv6_endpoint_type,notnull"`
	Name              string   `bun:"name,notnull"`
	Network           string   `bun:"network,notnull"`
	NetworkTier       string   `bun:"network_tier,notnull"`
	Subnetwork        string   `bun:"subnetwork,notnull"`
	PrefixLength      int      `bun:"prefix_length,notnull"`
	Purpose           string   `bun:"purpose,notnull"`
	SelfLink          string   `bun:"self_link,notnull"`
	Status            string   `bun:"status,notnull"`
	Project           *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// AddressToProject represents a link table connecting the [Project] with
// [Address] models.
type AddressToProject struct {
	bun.BaseModel `bun:"table:l_gcp_addr_to_project"`
	coremodels.Model

	ProjectID uint64 `bun:"project_id,notnull,unique:l_gcp_addr_to_project_key"`
	AddressID uint64 `bun:"address_id,notnull,unique:l_gcp_addr_to_project_key"`
}

// Subnet represents a GCP Subnet
type Subnet struct {
	bun.BaseModel `bun:"table:gcp_subnet"`
	coremodels.Model

	SubnetID          uint64   `bun:"subnet_id,notnull,unique:gcp_subnet_key"`
	VPCName           string   `bun:"vpc_name,notnull,unique:gcp_subnet_key"`
	ProjectID         string   `bun:"project_id,notnull,unique:gcp_subnet_key"`
	Name              string   `bun:"name,notnull"`
	Region            string   `bun:"region,notnull"`
	CreationTimestamp string   `bun:"creation_timestamp,nullzero"`
	Description       string   `bun:"description,notnull"`
	IPv4CIDRRange     string   `bun:"ipv4_cidr_range,notnull"`
	Gateway           net.IP   `bun:"gateway,nullzero,type:inet"`
	Purpose           string   `bun:"purpose,notnull"`
	Project           *Project `bun:"rel:has-one,join:project_id=project_id"`
	VPC               *VPC     `bun:"rel:has-one,join:vpc_name=name,join:project_id=project_id"`
}

// SubnetToVPC represents a link table connecting the [Subnet] with
// [VPC] models.
type SubnetToVPC struct {
	bun.BaseModel `bun:"table:l_gcp_subnet_to_vpc"`
	coremodels.Model

	VPCID    uint64 `bun:"vpc_id,notnull,unique:l_gcp_subnet_to_vpc_key"`
	SubnetID uint64 `bun:"subnet_id,notnull,unique:l_gcp_subnet_to_vpc_key"`
}

// SubnetToProject represents a link table connecting the [Subnet] with
// [Project] models.
type SubnetToProject struct {
	bun.BaseModel `bun:"table:l_gcp_subnet_to_project"`
	coremodels.Model

	ProjectID uint64 `bun:"project_id,notnull,unique:l_gcp_subnet_to_project_key"`
	SubnetID  uint64 `bun:"subnet_id,notnull,unique:l_gcp_subnet_to_project_key"`
}

// Bucket represents a GCP Bucket
type Bucket struct {
	bun.BaseModel `bun:"table:gcp_bucket"`
	coremodels.Model

	Name                string   `bun:"name,notnull,unique:gcp_bucket_key"`
	ProjectID           string   `bun:"project_id,notnull,unique:gcp_bucket_key"`
	LocationType        string   `bun:"location_type,notnull"`
	Location            string   `bun:"location,notnull"`
	DefaultStorageClass string   `bun:"default_storage_class,notnull"`
	CreationTimestamp   string   `bun:"creation_timestamp,nullzero"`
	Project             *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// ForwardingRule represents a GCP Forwarding Rule resource. The Forwarding
// Rules in GCP are global and regional. For more details please refer to the
// [Forwarding Rules overview] documentation.
//
// [Forwarding Rules overview]: https://cloud.google.com/load-balancing/docs/forwarding-rule-concepts
type ForwardingRule struct {
	bun.BaseModel `bun:"table:gcp_forwarding_rule"`
	coremodels.Model

	RuleID              uint64   `bun:"rule_id,notnull,unique:gcp_forwarding_rule_key"`
	ProjectID           string   `bun:"project_id,notnull,unique:gcp_forwarding_rule_key"`
	Name                string   `bun:"name,notnull"`
	IPAddress           net.IP   `bun:"ip_address,nullzero,type:inet"`
	IPProtocol          string   `bun:"ip_protocol,notnull"`
	IPVersion           string   `bun:"ip_version,notnull"`
	AllPorts            bool     `bun:"all_ports,notnull"`
	AllowGlobalAccess   bool     `bun:"allow_global_access,notnull"`
	BackendService      string   `bun:"backend_service,nullzero"`
	BaseForwardingRule  string   `bun:"base_forwarding_rule,nullzero"`
	CreationTimestamp   string   `bun:"creation_timestamp,nullzero"`
	Description         string   `bun:"description,notnull"`
	LoadBalancingScheme string   `bun:"load_balancing_scheme,notnull"`
	Network             string   `bun:"network,nullzero"`
	NetworkTier         string   `bun:"network_tier,nullzero"`
	PortRange           string   `bun:"port_range,nullzero"`
	Ports               []string `bun:"ports,nullzero,array"`
	Region              string   `bun:"region,notnull"`
	ServiceLabel        string   `bun:"service_label,nullzero"`
	ServiceName         string   `bun:"service_name,nullzero"`
	SourceIPRanges      []string `bun:"source_ip_ranges,nullzero,array"`
	Subnetwork          string   `bun:"subnetwork,nullzero"`
	Target              string   `bun:"target,nullzero"`
	Project             *Project `bun:"rel:has-one,join:project_id=project_id"`
	VPC                 *VPC     `bun:"rel:has-one,join:project_id=project_id,join:network=name"`
	Subnet              *Subnet  `bun:"rel:has-one,join:project_id=project_id,join:subnetwork=name"`
}

// ForwardingRuleToProject represents a link table connecting the
// [ForwardingRule] and [Project] models.
type ForwardingRuleToProject struct {
	bun.BaseModel `bun:"table:l_gcp_fr_to_project"`
	coremodels.Model

	RuleID    uint64 `bun:"rule_id,notnull,unique:l_gcp_fr_to_project_key"`
	ProjectID uint64 `bun:"project_id,notnull,unique:l_gcp_fr_to_project_key"`
}

// Disk represents a GCP Disk
type Disk struct {
	bun.BaseModel `bun:"table:gcp_disk"`
	coremodels.Model

	Name                string   `bun:"name,notnull,unique:gcp_disk_key"`
	ProjectID           string   `bun:"project_id,notnull,unique:gcp_disk_key"`
	Zone                string   `bun:"zone,notnull,unique:gcp_disk_key"`
	Region              string   `bun:"region,notnull"`
	Type                string   `bun:"type,notnull"`
	Description         string   `bun:"description,notnull"`
	IsRegional          bool     `bun:"is_regional,notnull"`
	CreationTimestamp   string   `bun:"creation_timestamp,nullzero"`
	LastAttachTimestamp string   `bun:"last_attach_timestamp,nullzero"`
	LastDetachTimestamp string   `bun:"last_detach_timestamp,nullzero"`
	SizeGB              int64    `bun:"size_gb,notnull"`
	Status              string   `bun:"status,nullzero"`
	Project             *Project `bun:"rel:has-one,join:project_id=project_id"`
}

// AttachedDisk represents an attached GCP Disk
type AttachedDisk struct {
	bun.BaseModel `bun:"table:gcp_attached_disk"`
	coremodels.Model

	InstanceName string    `bun:"instance_name,notnull,unique:gcp_attached_disk_key"`
	DiskName     string    `bun:"disk_name,notnull,unique:gcp_attached_disk_key"`
	ProjectID    string    `bun:"project_id,notnull,unique:gcp_attached_disk_key"`
	Zone         string    `bun:"zone,notnull"`
	Region       string    `bun:"region,notnull"`
	Instance     *Instance `bun:"rel:has-one,join:project_id=project_id,join:instance_name=name"`
	Disk         *Disk     `bun:"rel:has-one,join:project_id=project_id,join:disk_name=name"`
}

// InstanceToDisk represents a link table connecting the [Instance] with
// [Disk] models.
type InstanceToDisk struct {
	bun.BaseModel `bun:"table:l_gcp_instance_to_disk"`
	coremodels.Model

	InstanceID uint64 `bun:"instance_id,notnull,unique:l_gcp_instance_to_disk_key"`
	DiskID     uint64 `bun:"disk_id,notnull,unique:l_gcp_instance_to_disk_key"`
}

// GKECluster represents a GKE Cluster.
type GKECluster struct {
	bun.BaseModel `bun:"table:gcp_gke_cluster"`
	coremodels.Model

	Name                  string   `bun:"name,notnull"`
	ClusterID             string   `bun:"cluster_id,notnull,unique:gcp_gke_cluster_key"`
	ProjectID             string   `bun:"project_id,notnull,unique:gcp_gke_cluster_key"`
	Location              string   `bun:"location,notnull"`
	Network               string   `bun:"network,notnull"`
	Subnetwork            string   `bun:"subnetwork,notnull"`
	ClusterIPv4CIDR       string   `bun:"cluster_ipv4_cidr,notnull"`
	ServicesIPv4CIDR      string   `bun:"services_ipv4_cidr,notnull"`
	EnableKubernetesAlpha bool     `bun:"enable_k8s_alpha,notnull"`
	Endpoint              string   `bun:"endpoint,notnull"`
	InitialVersion        string   `bun:"initial_version,notnull"`
	CurrentMasterVersion  string   `bun:"current_master_version,notnull"`
	CAData                string   `bun:"ca_data,notnull"`
	Project               *Project `bun:"rel:has-one,join:project_id=project_id"`
	VPC                   *VPC     `bun:"rel:has-one,join:project_id=project_id,join:network=name"`
	Subnet                *Subnet  `bun:"rel:has-one,join:project_id=project_id,join:subnetwork=name,join:location=region"`
}

// GKEClusterToProject represents a link table connecting the [GKECluster] with
// [Project] models.
type GKEClusterToProject struct {
	bun.BaseModel `bun:"table:l_gcp_gke_cluster_to_project"`
	coremodels.Model

	ClusterID uint64 `bun:"cluster_id,notnull,unique:l_gcp_gke_cluster_to_project_key"`
	ProjectID uint64 `bun:"project_id,notnull,unique:l_gcp_gke_cluster_to_project_key"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("gcp:model:project", &Project{})
	registry.ModelRegistry.MustRegister("gcp:model:instance", &Instance{})
	registry.ModelRegistry.MustRegister("gcp:model:vpc", &VPC{})
	registry.ModelRegistry.MustRegister("gcp:model:address", &Address{})
	registry.ModelRegistry.MustRegister("gcp:model:nic", &NetworkInterface{})
	registry.ModelRegistry.MustRegister("gcp:model:subnet", &Subnet{})
	registry.ModelRegistry.MustRegister("gcp:model:bucket", &Bucket{})
	registry.ModelRegistry.MustRegister("gcp:model:forwarding_rule", &ForwardingRule{})
	registry.ModelRegistry.MustRegister("gcp:model:disk", &Disk{})
	registry.ModelRegistry.MustRegister("gcp:model:attached_disk", &AttachedDisk{})
	registry.ModelRegistry.MustRegister("gcp:model:gke_cluster", &GKECluster{})

	// Link tables
	registry.ModelRegistry.MustRegister("gcp:model:link_instance_to_project", &InstanceToProject{})
	registry.ModelRegistry.MustRegister("gcp:model:link_vpc_to_project", &VPCToProject{})
	registry.ModelRegistry.MustRegister("gcp:model:link_addr_to_project", &AddressToProject{})
	registry.ModelRegistry.MustRegister("gcp:model:link_instance_to_nic", &InstanceToNetworkInterface{})
	registry.ModelRegistry.MustRegister("gcp:model:link_subnet_to_vpc", &SubnetToVPC{})
	registry.ModelRegistry.MustRegister("gcp:model:link_subnet_to_project", &SubnetToProject{})
	registry.ModelRegistry.MustRegister("gcp:model:link_forwarding_rule_to_project", &ForwardingRuleToProject{})
	registry.ModelRegistry.MustRegister("gcp:model:link_instance_to_disk", &InstanceToDisk{})
	registry.ModelRegistry.MustRegister("gcp:model:link_gke_cluster_to_project", &GKEClusterToProject{})
}
