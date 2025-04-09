// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	coremodels "github.com/gardener/inventory/pkg/core/models"
	"github.com/gardener/inventory/pkg/core/registry"
)

// ShootToProject represents a link table connecting the Shoot with Project.
type ShootToProject struct {
	bun.BaseModel `bun:"table:l_g_shoot_to_project"`
	coremodels.Model

	ShootID   uuid.UUID `bun:"shoot_id,notnull,type:uuid,unique:l_g_shoot_to_project_key"`
	ProjectID uuid.UUID `bun:"project_id,notnull,type:uuid,unique:l_g_shoot_to_project_key"`
}

// ShootToSeed represents a link table connecting the Shoot with Seed.
type ShootToSeed struct {
	bun.BaseModel `bun:"table:l_g_shoot_to_seed"`
	coremodels.Model

	ShootID uuid.UUID `bun:"shoot_id,notnull,type:uuid,unique:l_g_shoot_to_seed_key"`
	SeedID  uuid.UUID `bun:"seed_id,notnull,type:uuid,unique:l_g_shoot_to_seed_key"`
}

// MachineToShoot represents a link table connecting the Machine with Shoot.
type MachineToShoot struct {
	bun.BaseModel `bun:"table:l_g_machine_to_shoot"`
	coremodels.Model

	ShootID   uuid.UUID `bun:"shoot_id,notnull,type:uuid,unique:l_g_machine_to_shoot_key"`
	MachineID uuid.UUID `bun:"machine_id,notnull,type:uuid,unique:l_g_machine_to_shoot_key"`
}

// Project represents a Gardener project
type Project struct {
	bun.BaseModel `bun:"table:g_project"`
	coremodels.Model

	Name              string           `bun:"name,notnull,unique"`
	Namespace         string           `bun:"namespace,notnull"`
	Status            string           `bun:"status,notnull"`
	Purpose           string           `bun:"purpose,notnull"`
	Owner             string           `bun:"owner,notnull"`
	CreationTimestamp time.Time        `bun:"creation_timestamp,nullzero"`
	Shoots            []*Shoot         `bun:"rel:has-many,join:name=project_name"`
	Members           []*ProjectMember `bun:"rel:has-many,join:name=project_name"`
}

// ProjectMember represents a member of a Gardener Project
type ProjectMember struct {
	bun.BaseModel `bun:"table:g_project_member"`
	coremodels.Model

	Name        string   `bun:"name,notnull,unique:g_project_member_key"`
	ProjectName string   `bun:"project_name,notnull,unique:g_project_member_key"`
	Kind        string   `bun:"kind,notnull"`
	Role        string   `bun:"role,notnull"`
	Project     *Project `bun:"rel:has-one,join:project_name=name"`
}

// ProjectToMember represents a link table connecting the [Project] and
// [ProjectMember] models.
type ProjectToMember struct {
	bun.BaseModel `bun:"table:l_g_project_to_member"`
	coremodels.Model

	ProjectID uuid.UUID `bun:"project_id,notnull,type:uuid,unique:l_g_project_to_member_key"`
	MemberID  uuid.UUID `bun:"member_id,notnull,type:uuid,unique:l_g_project_to_member_key"`
}

// Seed represents a Gardener seed
type Seed struct {
	bun.BaseModel `bun:"table:g_seed"`
	coremodels.Model

	Name              string     `bun:"name,notnull,unique"`
	KubernetesVersion string     `bun:"kubernetes_version,notnull"`
	CreationTimestamp time.Time  `bun:"creation_timestamp,nullzero"`
	Machines          []*Machine `bun:"rel:has-many,join:name=seed_name"`
	Shoots            []*Shoot   `bun:"rel:has-many,join:name=seed_name"`
}

// Shoot represents a Gardener shoot
type Shoot struct {
	bun.BaseModel `bun:"table:g_shoot"`
	coremodels.Model

	Name              string     `bun:"name,notnull"`
	TechnicalId       string     `bun:"technical_id,notnull,unique"`
	Namespace         string     `bun:"namespace,notnull"`
	ProjectName       string     `bun:"project_name,notnull"`
	CloudProfile      string     `bun:"cloud_profile,notnull"`
	Purpose           string     `bun:"purpose,notnull"`
	SeedName          string     `bun:"seed_name,notnull"`
	Status            string     `bun:"status,notnull"`
	IsHibernated      bool       `bun:"is_hibernated,notnull"`
	CreatedBy         string     `bun:"created_by,notnull"`
	Region            string     `bun:"region,nullzero"`
	KubernetesVersion string     `bun:"k8s_version,nullzero"`
	CreationTimestamp time.Time  `bun:"creation_timestamp,nullzero"`
	WorkerGroups      []string   `bun:"worker_groups,array,nullzero"`
	WorkerPrefixes    []string   `bun:"worker_prefixes,array,nullzero"`
	Seed              *Seed      `bun:"rel:has-one,join:seed_name=name"`
	Project           *Project   `bun:"rel:has-one,join:project_name=name"`
	Machines          []*Machine `bun:"rel:has-many,join:technical_id=namespace"`
}

// Machine represents a Gardener machine
type Machine struct {
	bun.BaseModel `bun:"table:g_machine"`
	coremodels.Model

	Name              string    `bun:"name,notnull,unique:g_machine_name_namespace_key"`
	Namespace         string    `bun:"namespace,notnull,unique:g_machine_name_namespace_key"`
	ProviderId        string    `bun:"provider_id,notnull"`
	Status            string    `bun:"status,notnull"`
	Node              string    `bun:"node,nullzero"`
	SeedName          string    `bun:"seed_name,notnull"`
	CreationTimestamp time.Time `bun:"creation_timestamp,nullzero"`
	Seed              *Seed     `bun:"rel:has-one,join:seed_name=name"`
	Shoot             *Shoot    `bun:"rel:has-one,join:namespace=technical_id"`
}

// BackupBucket represents a Gardener BackupBucket resource
type BackupBucket struct {
	bun.BaseModel `bun:"table:g_backup_bucket"`
	coremodels.Model

	Name              string    `bun:"name,notnull,unique"`
	ProviderType      string    `bun:"provider_type,notnull"`
	RegionName        string    `bun:"region_name,notnull"`
	State             string    `bun:"state,nullzero"`
	StateProgress     int       `bun:"state_progress,nullzero"`
	SeedName          string    `bun:"seed_name,notnull"`
	CreationTimestamp time.Time `bun:"creation_timestamp,nullzero"`
	Seed              *Seed     `bun:"rel:has-one,join:seed_name=name"`
}

// CloudProfile represents a Gardener CloudProfile resource
type CloudProfile struct {
	bun.BaseModel `bun:"table:g_cloud_profile"`
	coremodels.Model

	Name              string    `bun:"name,notnull,unique"`
	Type              string    `bun:"type,notnull"`
	CreationTimestamp time.Time `bun:"creation_timestamp,nullzero"`
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

	AWSImageID     uuid.UUID `bun:"aws_image_id,notnull,type:uuid,unique:l_g_aws_image_to_cloud_profile_key"`
	CloudProfileID uuid.UUID `bun:"cloud_profile_id,notnull,type:uuid,unique:l_g_aws_image_to_cloud_profile_key"`
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

	GCPImageID     uuid.UUID `bun:"gcp_image_id,notnull,type:uuid,unique:l_g_gcp_image_to_cloud_profile_key"`
	CloudProfileID uuid.UUID `bun:"cloud_profile_id,notnull,type:uuid,unique:l_g_gcp_image_to_cloud_profile_key"`
}

// CloudProfileAzureImage represents an Azure Machine Image collected from a CloudProfile.
type CloudProfileAzureImage struct {
	bun.BaseModel `bun:"table:g_cloud_profile_azure_image"`
	coremodels.Model

	Name             string        `bun:"name,notnull,unique:g_cloud_profile_azure_image_key"`
	Version          string        `bun:"version,notnull,unique:g_cloud_profile_azure_image_key"`
	Architecture     string        `bun:"architecture,notnull,unique:g_cloud_profile_azure_image_key"`
	CloudProfileName string        `bun:"cloud_profile_name,notnull,unique:g_cloud_profile_azure_image_key"`
	ImageID          string        `bun:"image_id,notnull,unique:g_cloud_profile_azure_image_key"`
	CloudProfile     *CloudProfile `bun:"rel:has-one,join:cloud_profile_name=name"`
}

// AzureImageToCloudProfile represents a link table connecting the CloudProfileAzureImage with CloudProfile.
type AzureImageToCloudProfile struct {
	bun.BaseModel `bun:"table:l_g_azure_image_to_cloud_profile"`
	coremodels.Model

	AzureImageID   uuid.UUID `bun:"azure_image_id,notnull,type:uuid,unique:l_g_azure_image_to_cloud_profile_key"`
	CloudProfileID uuid.UUID `bun:"cloud_profile_id,notnull,type:uuid,unique:l_g_azure_image_to_cloud_profile_key"`
}

// CloudProfileOpenStackImage represents an OpenStack Machine Image listed in a CloudProfile.
type CloudProfileOpenStackImage struct {
	bun.BaseModel `bun:"table:g_cloud_profile_openstack_image"`
	coremodels.Model

	Name             string        `bun:"name,notnull,unique:g_cloud_profile_openstack_image_key"`
	Version          string        `bun:"version,notnull,unique:g_cloud_profile_openstack_image_key"`
	RegionName       string        `bun:"region_name,notnull,unique:g_cloud_profile_openstack_image_key"`
	ImageID          string        `bun:"image_id,notnull,unique:g_cloud_profile_openstack_image_key"`
	Architecture     string        `bun:"architecture,notnull"`
	CloudProfileName string        `bun:"cloud_profile_name,notnull,unique:g_cloud_profile_openstack_image_key"`
	CloudProfile     *CloudProfile `bun:"rel:has-one,join:cloud_profile_name=name"`
}

// OpenStackImageToCloudProfile represents a link table connecting the CloudProfileOpenStackImage with CloudProfile.
type OpenStackImageToCloudProfile struct {
	bun.BaseModel `bun:"table:l_g_openstack_image_to_cloud_profile"`
	coremodels.Model

	OpenStackImageID uuid.UUID `bun:"openstack_image_id,notnull,type:uuid,unique:l_g_openstack_image_to_cloud_profile_key"`
	CloudProfileID   uuid.UUID `bun:"cloud_profile_id,notnull,type:uuid,unique:l_g_openstack_image_to_cloud_profile_key"`
}

// PersistentVolume represents a Kubernetes PV in Gardener
type PersistentVolume struct {
	bun.BaseModel `bun:"table:g_persistent_volume"`
	coremodels.Model

	Name              string    `bun:"name,notnull,unique:g_persistent_volume_key"`
	SeedName          string    `bun:"seed_name,notnull,unique:g_persistent_volume_key"`
	Provider          string    `bun:"provider,nullzero"`
	DiskRef           string    `bun:"disk_ref,nullzero"`
	Status            string    `bun:"status,notnull"`
	Capacity          string    `bun:"capacity,notnull"`
	StorageClass      string    `bun:"storage_class,notnull"`
	VolumeMode        string    `bun:"volume_mode,nullzero"`
	CreationTimestamp time.Time `bun:"creation_timestamp,nullzero"`
	Seed              *Seed     `bun:"rel:has-one,join:seed_name=name"`
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
	registry.ModelRegistry.MustRegister("g:model:persistent_volume", &PersistentVolume{})
	registry.ModelRegistry.MustRegister("g:model:project_member", &ProjectMember{})

	// Link tables
	registry.ModelRegistry.MustRegister("g:model:link_shoot_to_project", &ShootToProject{})
	registry.ModelRegistry.MustRegister("g:model:link_shoot_to_seed", &ShootToSeed{})
	registry.ModelRegistry.MustRegister("g:model:link_machine_to_shoot", &MachineToShoot{})
	registry.ModelRegistry.MustRegister("g:model:link_aws_image_to_cloud_profile", &AWSImageToCloudProfile{})
	registry.ModelRegistry.MustRegister("g:model:link_gcp_image_to_cloud_profile", &GCPImageToCloudProfile{})
	registry.ModelRegistry.MustRegister("g:model:link_azure_image_to_cloud_profile", &AzureImageToCloudProfile{})
	registry.ModelRegistry.MustRegister("g:model:link_project_to_member", &ProjectToMember{})
}
