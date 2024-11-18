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
	Region       string     `bun:"region,nullzero"`
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

	Name          string `bun:"name,notnull,unique"`
	ProviderType  string `bun:"provider_type,notnull"`
	RegionName    string `bun:"region_name,notnull"`
	State         string `bun:"state,nullzero"`
	StateProgress int    `bun:"state_progress,nullzero"`
	SeedName      string `bun:"seed_name,notnull"`
	Seed          *Seed  `bun:"rel:has-one,join:seed_name=name"`
}

// CloudProfile represents a Gardener CloudProfile resource
type CloudProfile struct {
	bun.BaseModel `bun:"table:g_cloud_profile"`
	coremodels.Model

	Name string `bun:"name,notnull,unique"`
	Type string `bun:"type,notnull"`
}

// CloudProfileAWSImage represents an AWS Machine Image collected from a CloudProfile.
// It is a separate resource to AMIs in the aws package, as we must match between
// what is required (this) and what is (AMIs)
type CloudProfileAWSImage struct {
	bun.BaseModel `bun:"table:g_cloud_profile_aws_image"`
	coremodels.Model

	Name             string        `bun:"name,notnull,unique:g_cloud_profile_aws_image_key"`
	Version          string        `bun:"version,notnull,unique:g_cloud_profile_aws_image_key"`
	RegionName       string        `bun:"region_name,notnull,unique:g_cloud_profile_aws_image_key"`
	AMI              string        `bun:"ami,notnull,unique:g_cloud_profile_aws_image_key"`
	Architecture     string        `bun:"architecture,notnull"`
	CloudProfileName string        `bun:"cloud_profile_name,notnull,unique:g_cloud_profile_aws_image_key"`
	CloudProfile     *CloudProfile `bun:"rel:has-one,join:cloud_profile_name=name"`
}

// AWSImageToCloudProfile represents a link table connecting the CloudProfileAWSImage with CloudProfile.
type AWSImageToCloudProfile struct {
	bun.BaseModel `bun:"table:l_g_aws_image_to_cloud_profile"`
	coremodels.Model

	AWSImageID     uint64 `bun:"aws_image_id,notnull,unique:l_g_aws_image_to_cloud_profile_key"`
	CloudProfileID uint64 `bun:"cloud_profile_id,notnull,unique:l_g_aws_image_to_cloud_profile_key"`
}

// CloudProfileGCPImage represents a GCP Machine Image collected from a CloudProfile.
type CloudProfileGCPImage struct {
	bun.BaseModel `bun:"table:g_cloud_profile_gcp_image"`
	coremodels.Model

	Name             string        `bun:"name,notnull,unique:g_cloud_profile_gcp_image_key"`
	Version          string        `bun:"version,notnull,unique:g_cloud_profile_gcp_image_key"`
	Image            string        `bun:"image,notnull,unique:g_cloud_profile_gcp_image_key"`
	Architecture     string        `bun:"architecture,notnull"`
	CloudProfileName string        `bun:"cloud_profile_name,notnull,unique:g_cloud_profile_gcp_image_key"`
	CloudProfile     *CloudProfile `bun:"rel:has-one,join:cloud_profile_name=name"`
}

// GCPImageToCloudProfile represents a link table connecting the CloudProfileGCPImage with CloudProfile.
type GCPImageToCloudProfile struct {
	bun.BaseModel `bun:"table:l_g_gcp_image_to_cloud_profile"`
	coremodels.Model

	GCPImageID     uint64 `bun:"gcp_image_id,notnull,unique:l_g_gcp_image_to_cloud_profile_key"`
	CloudProfileID uint64 `bun:"cloud_profile_id,notnull,unique:l_g_gcp_image_to_cloud_profile_key"`
}

// CloudProfileAzureImage represents an Azure Machine Image collected from a CloudProfile.
type CloudProfileAzureImage struct {
	bun.BaseModel `bun:"table:g_cloud_profile_azure_image"`
	coremodels.Model

	Name             string        `bun:"name,notnull,unique:g_cloud_profile_azure_image_key"`
	Version          string        `bun:"version,notnull,unique:g_cloud_profile_azure_image_key"`
	Architecture     string        `bun:"architecture,notnull,unique:g_cloud_profile_azure_image_key"`
	CloudProfileName string        `bun:"cloud_profile_name,notnull,unique:g_cloud_profile_azure_image_key"`
	URN              string        `bun:"urn,notnull"`
	GalleryImageID   string        `bun:"gallery_image_id,notnull"`
	CloudProfile     *CloudProfile `bun:"rel:has-one,join:cloud_profile_name=name"`
}

// AzureImageToCloudProfile represents a link table connecting the CloudProfileAzureImage with CloudProfile.
type AzureImageToCloudProfile struct {
	bun.BaseModel `bun:"table:l_g_azure_image_to_cloud_profile"`
	coremodels.Model

	AzureImageID   uint64 `bun:"azure_image_id,notnull,unique:l_g_azure_image_to_cloud_profile_key"`
	CloudProfileID uint64 `bun:"cloud_profile_id,notnull,unique:l_g_azure_image_to_cloud_profile_key"`
}

func init() {
	// Register the models with the default registry
	registry.ModelRegistry.MustRegister("g:model:project", &Project{})
	registry.ModelRegistry.MustRegister("g:model:seed", &Seed{})
	registry.ModelRegistry.MustRegister("g:model:shoot", &Shoot{})
	registry.ModelRegistry.MustRegister("g:model:machine", &Machine{})
	registry.ModelRegistry.MustRegister("g:model:backup_bucket", &BackupBucket{})
	registry.ModelRegistry.MustRegister("g:model:cloud_profile", &CloudProfile{})
	registry.ModelRegistry.MustRegister("g:model:cloud_profile_aws_image", &CloudProfileAWSImage{})
	registry.ModelRegistry.MustRegister("g:model:cloud_profile_gcp_image", &CloudProfileGCPImage{})
	registry.ModelRegistry.MustRegister("g:model:cloud_profile_azure_image", &CloudProfileAzureImage{})

	// Link tables
	registry.ModelRegistry.MustRegister("g:model:link_shoot_to_project", &ShootToProject{})
	registry.ModelRegistry.MustRegister("g:model:link_shoot_to_seed", &ShootToSeed{})
	registry.ModelRegistry.MustRegister("g:model:link_machine_to_shoot", &MachineToShoot{})
	registry.ModelRegistry.MustRegister("g:model:link_aws_image_to_cloud_profile", &AWSImageToCloudProfile{})
	registry.ModelRegistry.MustRegister("g:model:link_gcp_image_to_cloud_profile", &GCPImageToCloudProfile{})
	registry.ModelRegistry.MustRegister("g:model:link_azure_image_to_cloud_profile", &AzureImageToCloudProfile{})
}
