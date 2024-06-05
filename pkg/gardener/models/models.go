package models

import (
	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// Project represents a Gardener project
type Project struct {
	bun.BaseModel `bun:"table:g_project"`
	coremodels.Model

	Name      string `bun:"name,notnull,unique"`
	Namespace string `bun:"namespace,notnull"`
	Status    string `bun:"status,notnull"`
	Purpose   string `bun:"purpose,notnull"`
	Owner     string `bun:"owner,notnull"`
}

// Seed represents a Gardener seed
type Seed struct {
	bun.BaseModel `bun:"table:g_seed"`
	coremodels.Model

	Name              string `bun:"name,notnull,unique"`
	KubernetesVersion string `bun:"kubernetes_version,notnull"`
}

// Shoot represents a Gardener shoot
type Shoot struct {
	bun.BaseModel `bun:"table:g_shoot"`
	coremodels.Model

	Name         string `bun:"name,notnull"`
	TechnicalId  string `bun:"technical_id,notnull,unique"`
	Namespace    string `bun:"namespace,notnull"`
	ProjectName  string `bun:"project_name,notnull"`
	CloudProfile string `bun:"cloud_profile,notnull"`
	Purpose      string `bun:"purpose,notnull"`
	SeedName     string `bun:"seed_name,notnull"`
	Status       string `bun:"status,notnull"`
	IsHibernated bool   `bun:"is_hibernated,notnull"`
	CreatedBy    string `bun:"created_by,notnull"`
}

// Machine represents a Gardener machine
type Machine struct {
	bun.BaseModel `bun:"table:g_machine"`
	coremodels.Model

	Name       string `bun:"name,notnull,unique:g_machine_name_namespace_key"`
	Namespace  string `bun:"namespace,notnull,unique:g_machine_name_namespace_key"`
	ProviderId string `bun:"provider_id,notnull"`
	Status     string `bun:"status,notnull"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("g:model:project", &Project{})
	registry.ModelRegistry.MustRegister("g:model:seed", &Seed{})
	registry.ModelRegistry.MustRegister("g:model:shoot", &Shoot{})
	registry.ModelRegistry.MustRegister("g:model:machine", &Machine{})
}
