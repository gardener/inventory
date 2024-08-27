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
	CreationTimestamp    string   `bun:"creation_timestamp,notnull"`
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

	VPCID                uint64    `bun:"vpc_id,notnull,unique:gcp_vpc_key"`
	ProjectID            string    `bun:"project_id,notnull,unique:gcp_vpc_key"`
	Name                 string    `bun:"name,notnull,unique"`
	VPCCreationTimestamp time.Time `bun:"vpc_creation_timestamp"`
	Description          string    `bun:"description,nullzero"`
	GatewayIPv4          string    `bun:"gateway_ipv4,nullzero"`
	FirewallPolicy       string    `bun:"firewall_policy,nullzero"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("gcp:model:project", &Project{})
	registry.ModelRegistry.MustRegister("gcp:model:instance", &Instance{})
	registry.ModelRegistry.MustRegister("gcp:model:vpc", &VPC{})

	// Link tables
	registry.ModelRegistry.MustRegister("gcp:model:link_instance_to_project", &InstanceToProject{})
}
