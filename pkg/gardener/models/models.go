// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// ShootToProject represents a link table connecting the Shoot with Project.
type ShootToProject struct {
	bun.BaseModel `bun:"table:l_g_shoot_to_project"`
	coremodels.Model

	ShootID   uint64 `bun:"shoot_id,notnull,unique:l_g_shoot_to_project_key"`
	ProjectID uint64 `bun:"project_id,notnull,unique:l_g_shoot_to_project_key"`
}

// ShootToSeed represents a link table connecting the Shoot with Seed.
type ShootToSeed struct {
	bun.BaseModel `bun:"table:l_g_shoot_to_seed"`
	coremodels.Model

	ShootID uint64 `bun:"shoot_id,notnull,unique:l_g_shoot_to_seed_key"`
	SeedID  uint64 `bun:"seed_id,notnull,unique:l_g_shoot_to_seed_key"`
}

// MachineToShoot represents a link table connecting the Machine with Shoot.
type MachineToShoot struct {
	bun.BaseModel `bun:"table:l_g_machine_to_shoot"`
	coremodels.Model

	ShootID   uint64 `bun:"shoot_id,notnull,unique:l_g_machine_to_shoot_key"`
	MachineID uint64 `bun:"machine_id,notnull,unique:l_g_machine_to_shoot_key"`
}

// Project represents a Gardener project
type Project struct {
	bun.BaseModel `bun:"table:g_project"`
	coremodels.Model

	Name      string   `bun:"name,notnull,unique"`
	Namespace string   `bun:"namespace,notnull"`
	Status    string   `bun:"status,notnull"`
	Purpose   string   `bun:"purpose,notnull"`
	Owner     string   `bun:"owner,notnull"`
	Shoots    []*Shoot `bun:"rel:has-many,join:name=project_name"`
}

// Seed represents a Gardener seed
type Seed struct {
	bun.BaseModel `bun:"table:g_seed"`
	coremodels.Model

	Name              string   `bun:"name,notnull,unique"`
	KubernetesVersion string   `bun:"kubernetes_version,notnull"`
	Shoots            []*Shoot `bun:"rel:has-many,join:name=seed_name"`
}

// Shoot represents a Gardener shoot
type Shoot struct {
	bun.BaseModel `bun:"table:g_shoot"`
	coremodels.Model

	Name         string     `bun:"name,notnull"`
	TechnicalId  string     `bun:"technical_id,notnull,unique"`
	Namespace    string     `bun:"namespace,notnull"`
	ProjectName  string     `bun:"project_name,notnull"`
	CloudProfile string     `bun:"cloud_profile,notnull"`
	Purpose      string     `bun:"purpose,notnull"`
	SeedName     string     `bun:"seed_name,notnull"`
	Status       string     `bun:"status,notnull"`
	IsHibernated bool       `bun:"is_hibernated,notnull"`
	CreatedBy    string     `bun:"created_by,notnull"`
	Seed         *Seed      `bun:"rel:has-one,join:seed_name=name"`
	Project      *Project   `bun:"rel:has-one,join:project_name=name"`
	Machines     []*Machine `bun:"rel:has-many,join:technical_id=namespace"`
}

// Machine represents a Gardener machine
type Machine struct {
	bun.BaseModel `bun:"table:g_machine"`
	coremodels.Model

	Name       string `bun:"name,notnull,unique:g_machine_name_namespace_key"`
	Namespace  string `bun:"namespace,notnull,unique:g_machine_name_namespace_key"`
	ProviderId string `bun:"provider_id,notnull"`
	Status     string `bun:"status,notnull"`
	Shoot      *Shoot `bun:"rel:has-one,join:namespace=technical_id"`
}

// BackupBucket represents a Gardener BackupBucket resource
type BackupBucket struct {
	bun.BaseModel `bun:"table:g_backup_bucket"`
	coremodels.Model

	Name         string `bun:"name,notnull,unique"`
	ProviderType string `bun:"provider_type,notnull"`
	RegionName   string `bun:"region_name,notnull"`
	SeedName     string `bun:"seed_name,notnull"`
	Seed         *Seed  `bun:"rel:has-one,join:seed_name=name"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("g:model:project", &Project{})
	registry.ModelRegistry.MustRegister("g:model:seed", &Seed{})
	registry.ModelRegistry.MustRegister("g:model:shoot", &Shoot{})
	registry.ModelRegistry.MustRegister("g:model:machine", &Machine{})
	registry.ModelRegistry.MustRegister("g:model:backup_bucket", &BackupBucket{})

	// Link tables
	registry.ModelRegistry.MustRegister("g:model:link_shoot_to_project", &ShootToProject{})
	registry.ModelRegistry.MustRegister("g:model:link_shoot_to_seed", &ShootToSeed{})
	registry.ModelRegistry.MustRegister("g:model:link_machine_to_shoot", &MachineToShoot{})
}
