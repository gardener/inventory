--
-- Drop dependent views
--
DROP VIEW IF EXISTS aws_loadbalancer_interface;
DROP VIEW IF EXISTS aws_orphan_bucket;
DROP VIEW IF EXISTS aws_orphan_instance;
DROP VIEW IF EXISTS aws_unknown_instance_image;
DROP VIEW IF EXISTS aws_instance_interface;
DROP VIEW IF EXISTS aws_orphan_subnet;
DROP VIEW IF EXISTS aws_orphan_vpc;
DROP VIEW IF EXISTS gcp_orphan_disk;
DROP VIEW IF EXISTS gcp_zonal_disk;
DROP VIEW IF EXISTS gcp_regional_disk;
DROP VIEW IF EXISTS gcp_boot_disk;
DROP VIEW IF EXISTS gcp_data_disk;
DROP VIEW IF EXISTS gcp_orphan_instance;
DROP VIEW IF EXISTS gcp_orphan_subnet;
DROP VIEW IF EXISTS gcp_orphan_vpc;

--
-- Truncate link tables
--
TRUNCATE l_aws_image_to_region CASCADE;
TRUNCATE l_aws_instance_to_image CASCADE;
TRUNCATE l_aws_instance_to_net_interface CASCADE;
TRUNCATE l_aws_instance_to_region CASCADE;
TRUNCATE l_aws_instance_to_subnet CASCADE;
TRUNCATE l_aws_lb_to_net_interface CASCADE;
TRUNCATE l_aws_lb_to_region CASCADE;
TRUNCATE l_aws_lb_to_vpc CASCADE;
TRUNCATE l_aws_region_to_az CASCADE;
TRUNCATE l_aws_region_to_vpc CASCADE;
TRUNCATE l_aws_subnet_to_az CASCADE;
TRUNCATE l_aws_vpc_to_instance CASCADE;
TRUNCATE l_aws_vpc_to_subnet CASCADE;
TRUNCATE l_az_blob_container_to_rg CASCADE;
TRUNCATE l_az_lb_to_rg CASCADE;
TRUNCATE l_az_pub_addr_to_rg CASCADE;
TRUNCATE l_az_rg_to_subscription CASCADE;
TRUNCATE l_az_subnet_to_vpc CASCADE;
TRUNCATE l_az_vm_to_rg CASCADE;
TRUNCATE l_az_vpc_to_rg CASCADE;
TRUNCATE l_g_aws_image_to_cloud_profile CASCADE;
TRUNCATE l_g_azure_image_to_cloud_profile CASCADE;
TRUNCATE l_g_gcp_image_to_cloud_profile CASCADE;
TRUNCATE l_g_machine_to_shoot CASCADE;
TRUNCATE l_g_shoot_to_project CASCADE;
TRUNCATE l_g_shoot_to_seed CASCADE;
TRUNCATE l_gcp_addr_to_project CASCADE;
TRUNCATE l_gcp_fr_to_project CASCADE;
TRUNCATE l_gcp_gke_cluster_to_project CASCADE;
TRUNCATE l_gcp_instance_to_disk CASCADE;
TRUNCATE l_gcp_instance_to_nic CASCADE;
TRUNCATE l_gcp_instance_to_project CASCADE;
TRUNCATE l_gcp_subnet_to_project CASCADE;
TRUNCATE l_gcp_subnet_to_vpc CASCADE;
TRUNCATE l_gcp_vpc_to_project CASCADE;

--
-- Create BIGINT PKs on link tables
--
ALTER TABLE l_aws_image_to_region
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_image_to_region_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_instance_to_image
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_image_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_instance_to_net_interface
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_net_interface_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_instance_to_region
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_region_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_instance_to_subnet
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_subnet_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_lb_to_net_interface
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_lb_to_net_interface_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_lb_to_region
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_lb_to_region_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_lb_to_vpc
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_lb_to_vpc_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_region_to_az
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_region_to_az_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_region_to_vpc
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_region_to_vpc_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_subnet_to_az
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_subnet_to_az_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_vpc_to_instance
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_vpc_to_instance_pkey PRIMARY KEY (id);
ALTER TABLE l_aws_vpc_to_subnet
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_aws_vpc_to_subnet_pkey PRIMARY KEY (id);
ALTER TABLE l_az_blob_container_to_rg
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_az_blob_container_to_rg_pkey PRIMARY KEY (id);
ALTER TABLE l_az_lb_to_rg
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_az_lb_to_rg_pkey PRIMARY KEY (id);
ALTER TABLE l_az_pub_addr_to_rg
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_az_pub_addr_to_rg_pkey PRIMARY KEY (id);
ALTER TABLE l_az_rg_to_subscription
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_az_rg_to_subscription_pkey PRIMARY KEY (id);
ALTER TABLE l_az_subnet_to_vpc
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_az_subnet_to_vpc_pkey PRIMARY KEY (id);
ALTER TABLE l_az_vm_to_rg
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_az_vm_to_rg_pkey PRIMARY KEY (id);
ALTER TABLE l_az_vpc_to_rg
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_az_vpc_to_rg_pkey PRIMARY KEY (id);
ALTER TABLE l_g_aws_image_to_cloud_profile
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_g_aws_image_to_cloud_profile_pkey PRIMARY KEY (id);
ALTER TABLE l_g_azure_image_to_cloud_profile
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_g_azure_image_to_cloud_profile_pkey PRIMARY KEY (id);
ALTER TABLE l_g_gcp_image_to_cloud_profile
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_g_gcp_image_to_cloud_profile_pkey PRIMARY KEY (id);
ALTER TABLE l_g_machine_to_shoot
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_g_machine_to_shoot_pkey PRIMARY KEY (id);
ALTER TABLE l_g_shoot_to_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_g_shoot_to_project_pkey PRIMARY KEY (id);
ALTER TABLE l_g_shoot_to_seed
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_g_shoot_to_seed_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_addr_to_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_addr_to_project_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_fr_to_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_fr_to_project_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_gke_cluster_to_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_gke_cluster_to_project_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_instance_to_disk
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_instance_to_disk_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_instance_to_nic
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_instance_to_nic_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_instance_to_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_instance_to_project_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_subnet_to_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_subnet_to_project_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_subnet_to_vpc
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_subnet_to_vpc_pkey PRIMARY KEY (id);
ALTER TABLE l_gcp_vpc_to_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT l_gcp_vpc_to_project_pkey PRIMARY KEY (id);

--
-- Create BIGINT columns on link tables
--
ALTER TABLE l_aws_region_to_az
	DROP CONSTRAINT l_aws_region_to_az_key,
	DROP CONSTRAINT l_aws_region_to_az_region_id_fkey,
	DROP COLUMN region_id,
	ADD COLUMN region_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_region_to_az_az_id_fkey,
	DROP COLUMN az_id,
	ADD COLUMN az_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_region_to_az_key UNIQUE (region_id, az_id);
ALTER TABLE l_aws_region_to_vpc
	DROP CONSTRAINT l_aws_region_to_vpc_key,
	DROP CONSTRAINT l_aws_region_to_vpc_region_id_fkey,
	DROP COLUMN region_id,
	ADD COLUMN region_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_region_to_vpc_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_region_to_vpc_key UNIQUE (region_id, vpc_id);
ALTER TABLE l_aws_vpc_to_subnet
	DROP CONSTRAINT l_aws_vpc_to_subnet_key,
	DROP CONSTRAINT l_aws_vpc_to_subnet_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_vpc_to_subnet_subnet_id_fkey,
	DROP COLUMN subnet_id,
	ADD COLUMN subnet_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_vpc_to_subnet_key UNIQUE (vpc_id, subnet_id);
ALTER TABLE l_aws_vpc_to_instance
	DROP CONSTRAINT l_aws_vpc_to_instance_key,
	DROP CONSTRAINT l_aws_vpc_to_instance_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_vpc_to_instance_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_vpc_to_instance_key UNIQUE (vpc_id, instance_id);
ALTER TABLE l_aws_subnet_to_az
	DROP CONSTRAINT l_aws_subnet_to_az_key,
	DROP CONSTRAINT l_aws_subnet_to_az_az_id_fkey,
	DROP COLUMN az_id,
	ADD COLUMN az_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_subnet_to_az_subnet_id_fkey,
	DROP COLUMN subnet_id,
	ADD COLUMN subnet_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_subnet_to_az_key UNIQUE (az_id, subnet_id);
ALTER TABLE l_aws_instance_to_subnet
	DROP CONSTRAINT l_aws_instance_to_subnet_key,
	DROP CONSTRAINT l_aws_instance_to_subnet_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_instance_to_subnet_subnet_id_fkey,
	DROP COLUMN subnet_id,
	ADD COLUMN subnet_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_subnet_key UNIQUE (instance_id, subnet_id);
ALTER TABLE l_aws_instance_to_region
	DROP CONSTRAINT l_aws_instance_to_region_key,
	DROP CONSTRAINT l_aws_instance_to_region_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_instance_to_region_region_id_fkey,
	DROP COLUMN region_id,
	ADD COLUMN region_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_region_key UNIQUE (instance_id, region_id);
ALTER TABLE l_aws_instance_to_image
	DROP CONSTRAINT l_aws_instance_to_image_key,
	DROP CONSTRAINT l_aws_instance_to_image_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_instance_to_image_image_id_fkey,
	DROP COLUMN image_id,
	ADD COLUMN image_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_image_key UNIQUE (instance_id, image_id);
ALTER TABLE l_aws_image_to_region
	DROP CONSTRAINT l_aws_image_to_region_key,
	DROP CONSTRAINT l_aws_image_to_region_image_id_fkey,
	DROP COLUMN image_id,
	ADD COLUMN image_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_image_to_region_region_id_fkey,
	DROP COLUMN region_id,
	ADD COLUMN region_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_image_to_region_key UNIQUE (image_id, region_id);
ALTER TABLE l_aws_instance_to_net_interface
	DROP CONSTRAINT l_aws_instance_to_net_interface_key,
	DROP CONSTRAINT l_aws_instance_to_net_interface_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_instance_to_net_interface_ni_id_fkey,
	DROP COLUMN ni_id,
	ADD COLUMN ni_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_instance_to_net_interface_key UNIQUE (instance_id, ni_id);
ALTER TABLE l_aws_lb_to_vpc
	DROP CONSTRAINT l_aws_lb_to_vpc_key,
	DROP CONSTRAINT l_aws_lb_to_vpc_lb_id_fkey,
	DROP COLUMN lb_id,
	ADD COLUMN lb_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_lb_to_vpc_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_lb_to_vpc_key UNIQUE (lb_id, vpc_id);
ALTER TABLE l_aws_lb_to_region
	DROP CONSTRAINT l_aws_lb_to_region_key,
	DROP CONSTRAINT l_aws_lb_to_region_lb_id_fkey,
	DROP COLUMN lb_id,
	ADD COLUMN lb_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_lb_to_region_region_id_fkey,
	DROP COLUMN region_id,
	ADD COLUMN region_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_lb_to_region_key UNIQUE (lb_id, region_id);
ALTER TABLE l_aws_lb_to_net_interface
	DROP CONSTRAINT l_aws_lb_to_net_interface_key,
	DROP CONSTRAINT l_aws_lb_to_net_interface_lb_id_fkey,
	DROP COLUMN lb_id,
	ADD COLUMN lb_id BIGINT NOT NULL,
	DROP CONSTRAINT l_aws_lb_to_net_interface_ni_id_fkey,
	DROP COLUMN ni_id,
	ADD COLUMN ni_id BIGINT NOT NULL,
	ADD CONSTRAINT l_aws_lb_to_net_interface_key UNIQUE (lb_id, ni_id);
ALTER TABLE l_g_shoot_to_project
	DROP CONSTRAINT l_g_shoot_to_project_key,
	DROP CONSTRAINT l_g_shoot_to_project_shoot_id_fkey,
	DROP COLUMN shoot_id,
	ADD COLUMN shoot_id BIGINT NOT NULL,
	DROP CONSTRAINT l_g_shoot_to_project_project_id_fkey,
	DROP COLUMN project_id,
	ADD COLUMN project_id BIGINT NOT NULL,
	ADD CONSTRAINT l_g_shoot_to_project_key UNIQUE (shoot_id, project_id);
ALTER TABLE l_g_shoot_to_seed
	DROP CONSTRAINT l_g_shoot_to_seed_key,
	DROP CONSTRAINT l_g_shoot_to_seed_shoot_id_fkey,
	DROP COLUMN shoot_id,
	ADD COLUMN shoot_id BIGINT NOT NULL,
	DROP CONSTRAINT l_g_shoot_to_seed_seed_id_fkey,
	DROP COLUMN seed_id,
	ADD COLUMN seed_id BIGINT NOT NULL,
	ADD CONSTRAINT l_g_shoot_to_seed_key UNIQUE (shoot_id, seed_id);
ALTER TABLE l_g_machine_to_shoot
	DROP CONSTRAINT l_g_machine_to_shoot_key,
	DROP CONSTRAINT l_g_machine_to_shoot_shoot_id_fkey,
	DROP COLUMN shoot_id,
	ADD COLUMN shoot_id BIGINT NOT NULL,
	DROP CONSTRAINT l_g_machine_to_shoot_machine_id_fkey,
	DROP COLUMN machine_id,
	ADD COLUMN machine_id BIGINT NOT NULL,
	ADD CONSTRAINT l_g_machine_to_shoot_key UNIQUE (shoot_id, machine_id);
ALTER TABLE l_g_aws_image_to_cloud_profile
	DROP CONSTRAINT l_g_aws_image_to_cloud_profile_key,
	DROP CONSTRAINT l_g_aws_image_to_cloud_profile_aws_image_id_fkey,
	DROP COLUMN aws_image_id,
	ADD COLUMN aws_image_id BIGINT NOT NULL,
	DROP CONSTRAINT l_g_aws_image_to_cloud_profile_cloud_profile_id_fkey,
	DROP COLUMN cloud_profile_id,
	ADD COLUMN cloud_profile_id BIGINT NOT NULL,
	ADD CONSTRAINT l_g_aws_image_to_cloud_profile_key UNIQUE (aws_image_id, cloud_profile_id);
ALTER TABLE l_g_gcp_image_to_cloud_profile
	DROP CONSTRAINT l_g_gcp_image_to_cloud_profile_key,
	DROP CONSTRAINT l_g_gcp_image_to_cloud_profile_gcp_image_id_fkey,
	DROP COLUMN gcp_image_id,
	ADD COLUMN gcp_image_id BIGINT NOT NULL,
	DROP CONSTRAINT l_g_gcp_image_to_cloud_profile_cloud_profile_id_fkey,
	DROP COLUMN cloud_profile_id,
	ADD COLUMN cloud_profile_id BIGINT NOT NULL,
	ADD CONSTRAINT l_g_gcp_image_to_cloud_profile_key UNIQUE (gcp_image_id, cloud_profile_id);
ALTER TABLE l_g_azure_image_to_cloud_profile
	DROP CONSTRAINT l_g_azure_image_to_cloud_profile_key,
	DROP CONSTRAINT l_g_azure_image_to_cloud_profile_azure_image_id_fkey,
	DROP COLUMN azure_image_id,
	ADD COLUMN azure_image_id BIGINT NOT NULL,
	DROP CONSTRAINT l_g_azure_image_to_cloud_profile_cloud_profile_id_fkey,
	DROP COLUMN cloud_profile_id,
	ADD COLUMN cloud_profile_id BIGINT NOT NULL,
	ADD CONSTRAINT l_g_azure_image_to_cloud_profile_key UNIQUE (azure_image_id, cloud_profile_id);
ALTER TABLE l_gcp_instance_to_nic
	DROP CONSTRAINT l_gcp_instance_to_nic_key,
	DROP CONSTRAINT l_gcp_instance_to_nic_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_instance_to_nic_nic_id_fkey,
	DROP COLUMN nic_id,
	ADD COLUMN nic_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_instance_to_nic_key UNIQUE (instance_id, nic_id);
ALTER TABLE l_gcp_instance_to_project
	DROP CONSTRAINT l_gcp_instance_to_project_key,
	DROP CONSTRAINT l_gcp_instance_to_project_project_id_fkey,
	DROP COLUMN project_id,
	ADD COLUMN project_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_instance_to_project_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_instance_to_project_key UNIQUE (project_id, instance_id);
ALTER TABLE l_gcp_vpc_to_project
	DROP CONSTRAINT l_gcp_vpc_to_project_key,
	DROP CONSTRAINT l_gcp_vpc_to_project_project_id_fkey,
	DROP COLUMN project_id,
	ADD COLUMN project_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_vpc_to_project_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_vpc_to_project_key UNIQUE (project_id, vpc_id);
ALTER TABLE l_gcp_addr_to_project
	DROP CONSTRAINT l_gcp_addr_to_project_key,
	DROP CONSTRAINT l_gcp_addr_to_project_project_id_fkey,
	DROP COLUMN project_id,
	ADD COLUMN project_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_addr_to_project_address_id_fkey,
	DROP COLUMN address_id,
	ADD COLUMN address_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_addr_to_project_key UNIQUE (project_id, address_id);
ALTER TABLE l_gcp_subnet_to_vpc
	DROP CONSTRAINT l_gcp_subnet_to_vpc_key,
	DROP CONSTRAINT l_gcp_subnet_to_vpc_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_subnet_to_vpc_subnet_id_fkey,
	DROP COLUMN subnet_id,
	ADD COLUMN subnet_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_subnet_to_vpc_key UNIQUE (vpc_id, subnet_id);
ALTER TABLE l_gcp_subnet_to_project
	DROP CONSTRAINT l_gcp_subnet_to_project_key,
	DROP CONSTRAINT l_gcp_subnet_to_project_project_id_fkey,
	DROP COLUMN project_id,
	ADD COLUMN project_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_subnet_to_project_subnet_id_fkey,
	DROP COLUMN subnet_id,
	ADD COLUMN subnet_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_subnet_to_project_key UNIQUE (project_id, subnet_id);
ALTER TABLE l_gcp_fr_to_project
	DROP CONSTRAINT l_gcp_fr_to_project_key,
	DROP CONSTRAINT l_gcp_fr_to_project_rule_id_fkey,
	DROP COLUMN rule_id,
	ADD COLUMN rule_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_fr_to_project_project_id_fkey,
	DROP COLUMN project_id,
	ADD COLUMN project_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_fr_to_project_key UNIQUE (rule_id, project_id);
ALTER TABLE l_gcp_instance_to_disk
	DROP CONSTRAINT l_gcp_instance_to_disk_key,
	DROP CONSTRAINT l_gcp_instance_to_disk_instance_id_fkey,
	DROP COLUMN instance_id,
	ADD COLUMN instance_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_instance_to_disk_disk_id_fkey,
	DROP COLUMN disk_id,
	ADD COLUMN disk_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_instance_to_disk_key UNIQUE (instance_id, disk_id);
ALTER TABLE l_gcp_gke_cluster_to_project
	DROP CONSTRAINT l_gcp_gke_cluster_to_project_key,
	DROP CONSTRAINT l_gcp_gke_cluster_to_project_cluster_id_fkey,
	DROP COLUMN cluster_id,
	ADD COLUMN cluster_id BIGINT NOT NULL,
	DROP CONSTRAINT l_gcp_gke_cluster_to_project_project_id_fkey,
	DROP COLUMN project_id,
	ADD COLUMN project_id BIGINT NOT NULL,
	ADD CONSTRAINT l_gcp_gke_cluster_to_project_key UNIQUE (cluster_id, project_id);
ALTER TABLE l_az_rg_to_subscription
	DROP CONSTRAINT l_az_rg_to_subscription_key,
	DROP CONSTRAINT l_az_rg_to_subscription_rg_id_fkey,
	DROP COLUMN rg_id,
	ADD COLUMN rg_id BIGINT NOT NULL,
	DROP CONSTRAINT l_az_rg_to_subscription_sub_id_fkey,
	DROP COLUMN sub_id,
	ADD COLUMN sub_id BIGINT NOT NULL,
	ADD CONSTRAINT l_az_rg_to_subscription_key UNIQUE (rg_id, sub_id);
ALTER TABLE l_az_vm_to_rg
	DROP CONSTRAINT l_az_vm_to_rg_key,
	DROP CONSTRAINT l_az_vm_to_rg_rg_id_fkey,
	DROP COLUMN rg_id,
	ADD COLUMN rg_id BIGINT NOT NULL,
	DROP CONSTRAINT l_az_vm_to_rg_vm_id_fkey,
	DROP COLUMN vm_id,
	ADD COLUMN vm_id BIGINT NOT NULL,
	ADD CONSTRAINT l_az_vm_to_rg_key UNIQUE (rg_id, vm_id);
ALTER TABLE l_az_pub_addr_to_rg
	DROP CONSTRAINT l_az_pub_addr_to_rg_key,
	DROP CONSTRAINT l_az_pub_addr_to_rg_rg_id_fkey,
	DROP COLUMN rg_id,
	ADD COLUMN rg_id BIGINT NOT NULL,
	DROP CONSTRAINT l_az_pub_addr_to_rg_pa_id_fkey,
	DROP COLUMN pa_id,
	ADD COLUMN pa_id BIGINT NOT NULL,
	ADD CONSTRAINT l_az_pub_addr_to_rg_key UNIQUE (rg_id, pa_id);
ALTER TABLE l_az_lb_to_rg
	DROP CONSTRAINT l_az_lb_to_rg_key,
	DROP CONSTRAINT l_az_lb_to_rg_rg_id_fkey,
	DROP COLUMN rg_id,
	ADD COLUMN rg_id BIGINT NOT NULL,
	DROP CONSTRAINT l_az_lb_to_rg_lb_id_fkey,
	DROP COLUMN lb_id,
	ADD COLUMN lb_id BIGINT NOT NULL,
	ADD CONSTRAINT l_az_lb_to_rg_key UNIQUE (rg_id, lb_id);
ALTER TABLE l_az_vpc_to_rg
	DROP CONSTRAINT l_az_vpc_to_rg_key,
	DROP CONSTRAINT l_az_vpc_to_rg_rg_id_fkey,
	DROP COLUMN rg_id,
	ADD COLUMN rg_id BIGINT NOT NULL,
	DROP CONSTRAINT l_az_vpc_to_rg_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	ADD CONSTRAINT l_az_vpc_to_rg_key UNIQUE (rg_id, vpc_id);
ALTER TABLE l_az_subnet_to_vpc
	DROP CONSTRAINT l_az_subnet_to_vpc_key,
	DROP CONSTRAINT l_az_subnet_to_vpc_subnet_id_fkey,
	DROP COLUMN subnet_id,
	ADD COLUMN subnet_id BIGINT NOT NULL,
	DROP CONSTRAINT l_az_subnet_to_vpc_vpc_id_fkey,
	DROP COLUMN vpc_id,
	ADD COLUMN vpc_id BIGINT NOT NULL,
	ADD CONSTRAINT l_az_subnet_to_vpc_key UNIQUE (subnet_id, vpc_id);
ALTER TABLE l_az_blob_container_to_rg
	DROP CONSTRAINT l_az_blob_container_to_rg_key,
	DROP CONSTRAINT l_az_blob_container_to_rg_blob_container_id_fkey,
	DROP COLUMN blob_container_id,
	ADD COLUMN blob_container_id BIGINT NOT NULL,
	DROP CONSTRAINT l_az_blob_container_to_rg_rg_id_fkey,
	DROP COLUMN rg_id,
	ADD COLUMN rg_id BIGINT NOT NULL,
	ADD CONSTRAINT l_az_blob_container_to_rg_key UNIQUE (blob_container_id, rg_id);

--
-- Create BIGINT PKs on non-link tables
--
ALTER TABLE aws_az
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_az_pkey PRIMARY KEY (id);
ALTER TABLE aws_bucket
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_bucket_pkey PRIMARY KEY (id);
ALTER TABLE aws_geodata
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_geodata_pkey PRIMARY KEY (id);
ALTER TABLE aws_image
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_image_pkey PRIMARY KEY (id);
ALTER TABLE aws_instance
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_instance_pkey PRIMARY KEY (id);
ALTER TABLE aws_loadbalancer
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_loadbalancer_pkey PRIMARY KEY (id);
ALTER TABLE aws_net_interface
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_net_interface_pkey PRIMARY KEY (id);
ALTER TABLE aws_region
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_region_pkey PRIMARY KEY (id);
ALTER TABLE aws_subnet
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_subnet_pkey PRIMARY KEY (id);
ALTER TABLE aws_vpc
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT aws_vpc_pkey PRIMARY KEY (id);
ALTER TABLE az_blob_container
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_blob_container_pkey PRIMARY KEY (id);
ALTER TABLE az_lb
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_lb_pkey PRIMARY KEY (id);
ALTER TABLE az_public_address
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_public_address_pkey PRIMARY KEY (id);
ALTER TABLE az_resource_group
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_resource_group_pkey PRIMARY KEY (id);
ALTER TABLE az_storage_account
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_storage_account_pkey PRIMARY KEY (id);
ALTER TABLE az_subnet
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_subnet_pkey PRIMARY KEY (id);
ALTER TABLE az_subscription
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_subscription_pkey PRIMARY KEY (id);
ALTER TABLE az_vm
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_vm_pkey PRIMARY KEY (id);
ALTER TABLE az_vpc
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT az_vpc_pkey PRIMARY KEY (id);
ALTER TABLE g_backup_bucket
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_backup_bucket_pkey PRIMARY KEY (id);
ALTER TABLE g_cloud_profile
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_cloud_profile_pkey PRIMARY KEY (id);
ALTER TABLE g_cloud_profile_aws_image
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_cloud_profile_aws_image_pkey PRIMARY KEY (id);
ALTER TABLE g_cloud_profile_azure_image
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_cloud_profile_azure_image_pkey PRIMARY KEY (id);
ALTER TABLE g_cloud_profile_gcp_image
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_cloud_profile_gcp_image_pkey PRIMARY KEY (id);
ALTER TABLE g_machine
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_machine_pkey PRIMARY KEY (id);
ALTER TABLE g_persistent_volume
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_persistent_volume_pkey PRIMARY KEY (id);
ALTER TABLE g_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_project_pkey PRIMARY KEY (id);
ALTER TABLE g_seed
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_seed_pkey PRIMARY KEY (id);
ALTER TABLE g_shoot
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT g_shoot_pkey PRIMARY KEY (id);
ALTER TABLE gcp_address
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_address_pkey PRIMARY KEY (id);
ALTER TABLE gcp_attached_disk
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_attached_disk_pkey PRIMARY KEY (id);
ALTER TABLE gcp_bucket
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_bucket_pkey PRIMARY KEY (id);
ALTER TABLE gcp_disk
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_disk_pkey PRIMARY KEY (id);
ALTER TABLE gcp_forwarding_rule
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_forwarding_rule_pkey PRIMARY KEY (id);
ALTER TABLE gcp_gke_cluster
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_gke_cluster_pkey PRIMARY KEY (id);
ALTER TABLE gcp_instance
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_instance_pkey PRIMARY KEY (id);
ALTER TABLE gcp_nic
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_nic_pkey PRIMARY KEY (id);
ALTER TABLE gcp_project
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_project_pkey PRIMARY KEY (id);
ALTER TABLE gcp_subnet
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_subnet_pkey PRIMARY KEY (id);
ALTER TABLE gcp_vpc
	DROP COLUMN id,
	ADD COLUMN id BIGSERIAL NOT NULL,
	ADD CONSTRAINT gcp_vpc_pkey PRIMARY KEY (id);

--
-- Create foreign key constraints on link tables
--
ALTER TABLE l_aws_region_to_az ADD CONSTRAINT l_aws_region_to_az_region_id_fkey FOREIGN KEY (region_id) REFERENCES aws_region (id) ON DELETE CASCADE;
ALTER TABLE l_aws_region_to_az ADD CONSTRAINT l_aws_region_to_az_az_id_fkey FOREIGN KEY (az_id) REFERENCES aws_az (id) ON DELETE CASCADE;
ALTER TABLE l_aws_region_to_vpc ADD CONSTRAINT l_aws_region_to_vpc_region_id_fkey FOREIGN KEY (region_id) REFERENCES aws_region (id) ON DELETE CASCADE;
ALTER TABLE l_aws_region_to_vpc ADD CONSTRAINT l_aws_region_to_vpc_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES aws_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_aws_vpc_to_subnet ADD CONSTRAINT l_aws_vpc_to_subnet_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES aws_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_aws_vpc_to_subnet ADD CONSTRAINT l_aws_vpc_to_subnet_subnet_id_fkey FOREIGN KEY (subnet_id) REFERENCES aws_subnet (id) ON DELETE CASCADE;
ALTER TABLE l_aws_vpc_to_instance ADD CONSTRAINT l_aws_vpc_to_instance_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES aws_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_aws_vpc_to_instance ADD CONSTRAINT l_aws_vpc_to_instance_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES aws_instance (id) ON DELETE CASCADE;
ALTER TABLE l_aws_subnet_to_az ADD CONSTRAINT l_aws_subnet_to_az_az_id_fkey FOREIGN KEY (az_id) REFERENCES aws_az (id) ON DELETE CASCADE;
ALTER TABLE l_aws_subnet_to_az ADD CONSTRAINT l_aws_subnet_to_az_subnet_id_fkey FOREIGN KEY (subnet_id) REFERENCES aws_subnet (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_subnet ADD CONSTRAINT l_aws_instance_to_subnet_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES aws_instance (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_subnet ADD CONSTRAINT l_aws_instance_to_subnet_subnet_id_fkey FOREIGN KEY (subnet_id) REFERENCES aws_subnet (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_region ADD CONSTRAINT l_aws_instance_to_region_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES aws_instance (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_region ADD CONSTRAINT l_aws_instance_to_region_region_id_fkey FOREIGN KEY (region_id) REFERENCES aws_region (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_image ADD CONSTRAINT l_aws_instance_to_image_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES aws_instance (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_image ADD CONSTRAINT l_aws_instance_to_image_image_id_fkey FOREIGN KEY (image_id) REFERENCES aws_image (id) ON DELETE CASCADE;
ALTER TABLE l_aws_image_to_region ADD CONSTRAINT l_aws_image_to_region_image_id_fkey FOREIGN KEY (image_id) REFERENCES aws_image (id) ON DELETE CASCADE;
ALTER TABLE l_aws_image_to_region ADD CONSTRAINT l_aws_image_to_region_region_id_fkey FOREIGN KEY (region_id) REFERENCES aws_region (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_net_interface ADD CONSTRAINT l_aws_instance_to_net_interface_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES aws_instance (id) ON DELETE CASCADE;
ALTER TABLE l_aws_instance_to_net_interface ADD CONSTRAINT l_aws_instance_to_net_interface_ni_id_fkey FOREIGN KEY (ni_id) REFERENCES aws_net_interface (id) ON DELETE CASCADE;
ALTER TABLE l_aws_lb_to_vpc ADD CONSTRAINT l_aws_lb_to_vpc_lb_id_fkey FOREIGN KEY (lb_id) REFERENCES aws_loadbalancer (id) ON DELETE CASCADE;
ALTER TABLE l_aws_lb_to_vpc ADD CONSTRAINT l_aws_lb_to_vpc_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES aws_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_aws_lb_to_region ADD CONSTRAINT l_aws_lb_to_region_lb_id_fkey FOREIGN KEY (lb_id) REFERENCES aws_loadbalancer (id) ON DELETE CASCADE;
ALTER TABLE l_aws_lb_to_region ADD CONSTRAINT l_aws_lb_to_region_region_id_fkey FOREIGN KEY (region_id) REFERENCES aws_region (id) ON DELETE CASCADE;
ALTER TABLE l_aws_lb_to_net_interface ADD CONSTRAINT l_aws_lb_to_net_interface_lb_id_fkey FOREIGN KEY (lb_id) REFERENCES aws_loadbalancer (id) ON DELETE CASCADE;
ALTER TABLE l_aws_lb_to_net_interface ADD CONSTRAINT l_aws_lb_to_net_interface_ni_id_fkey FOREIGN KEY (ni_id) REFERENCES aws_net_interface (id) ON DELETE CASCADE;
ALTER TABLE l_g_shoot_to_project ADD CONSTRAINT l_g_shoot_to_project_shoot_id_fkey FOREIGN KEY (shoot_id) REFERENCES g_shoot (id) ON DELETE CASCADE;
ALTER TABLE l_g_shoot_to_project ADD CONSTRAINT l_g_shoot_to_project_project_id_fkey FOREIGN KEY (project_id) REFERENCES g_project (id) ON DELETE CASCADE;
ALTER TABLE l_g_shoot_to_seed ADD CONSTRAINT l_g_shoot_to_seed_shoot_id_fkey FOREIGN KEY (shoot_id) REFERENCES g_shoot (id) ON DELETE CASCADE;
ALTER TABLE l_g_shoot_to_seed ADD CONSTRAINT l_g_shoot_to_seed_seed_id_fkey FOREIGN KEY (seed_id) REFERENCES g_seed (id) ON DELETE CASCADE;
ALTER TABLE l_g_machine_to_shoot ADD CONSTRAINT l_g_machine_to_shoot_shoot_id_fkey FOREIGN KEY (shoot_id) REFERENCES g_shoot (id) ON DELETE CASCADE;
ALTER TABLE l_g_machine_to_shoot ADD CONSTRAINT l_g_machine_to_shoot_machine_id_fkey FOREIGN KEY (machine_id) REFERENCES g_machine (id) ON DELETE CASCADE;
ALTER TABLE l_g_aws_image_to_cloud_profile ADD CONSTRAINT l_g_aws_image_to_cloud_profile_aws_image_id_fkey FOREIGN KEY (aws_image_id) REFERENCES g_cloud_profile_aws_image (id) ON DELETE CASCADE;
ALTER TABLE l_g_aws_image_to_cloud_profile ADD CONSTRAINT l_g_aws_image_to_cloud_profile_cloud_profile_id_fkey FOREIGN KEY (cloud_profile_id) REFERENCES g_cloud_profile (id) ON DELETE CASCADE;
ALTER TABLE l_g_gcp_image_to_cloud_profile ADD CONSTRAINT l_g_gcp_image_to_cloud_profile_gcp_image_id_fkey FOREIGN KEY (gcp_image_id) REFERENCES g_cloud_profile_gcp_image (id) ON DELETE CASCADE;
ALTER TABLE l_g_gcp_image_to_cloud_profile ADD CONSTRAINT l_g_gcp_image_to_cloud_profile_cloud_profile_id_fkey FOREIGN KEY (cloud_profile_id) REFERENCES g_cloud_profile (id) ON DELETE CASCADE;
ALTER TABLE l_g_azure_image_to_cloud_profile ADD CONSTRAINT l_g_azure_image_to_cloud_profile_azure_image_id_fkey FOREIGN KEY (azure_image_id) REFERENCES g_cloud_profile_azure_image (id) ON DELETE CASCADE;
ALTER TABLE l_g_azure_image_to_cloud_profile ADD CONSTRAINT l_g_azure_image_to_cloud_profile_cloud_profile_id_fkey FOREIGN KEY (cloud_profile_id) REFERENCES g_cloud_profile (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_instance_to_nic ADD CONSTRAINT l_gcp_instance_to_nic_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES gcp_instance (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_instance_to_nic ADD CONSTRAINT l_gcp_instance_to_nic_nic_id_fkey FOREIGN KEY (nic_id) REFERENCES gcp_nic (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_instance_to_project ADD CONSTRAINT l_gcp_instance_to_project_project_id_fkey FOREIGN KEY (project_id) REFERENCES gcp_project (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_instance_to_project ADD CONSTRAINT l_gcp_instance_to_project_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES gcp_instance (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_vpc_to_project ADD CONSTRAINT l_gcp_vpc_to_project_project_id_fkey FOREIGN KEY (project_id) REFERENCES gcp_project (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_vpc_to_project ADD CONSTRAINT l_gcp_vpc_to_project_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES gcp_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_addr_to_project ADD CONSTRAINT l_gcp_addr_to_project_project_id_fkey FOREIGN KEY (project_id) REFERENCES gcp_project (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_addr_to_project ADD CONSTRAINT l_gcp_addr_to_project_address_id_fkey FOREIGN KEY (address_id) REFERENCES gcp_address (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_subnet_to_vpc ADD CONSTRAINT l_gcp_subnet_to_vpc_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES gcp_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_subnet_to_vpc ADD CONSTRAINT l_gcp_subnet_to_vpc_subnet_id_fkey FOREIGN KEY (subnet_id) REFERENCES gcp_subnet (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_subnet_to_project ADD CONSTRAINT l_gcp_subnet_to_project_project_id_fkey FOREIGN KEY (project_id) REFERENCES gcp_project (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_subnet_to_project ADD CONSTRAINT l_gcp_subnet_to_project_subnet_id_fkey FOREIGN KEY (subnet_id) REFERENCES gcp_subnet (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_fr_to_project ADD CONSTRAINT l_gcp_fr_to_project_rule_id_fkey FOREIGN KEY (rule_id) REFERENCES gcp_forwarding_rule (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_fr_to_project ADD CONSTRAINT l_gcp_fr_to_project_project_id_fkey FOREIGN KEY (project_id) REFERENCES gcp_project (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_instance_to_disk ADD CONSTRAINT l_gcp_instance_to_disk_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES gcp_instance (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_instance_to_disk ADD CONSTRAINT l_gcp_instance_to_disk_disk_id_fkey FOREIGN KEY (disk_id) REFERENCES gcp_disk (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_gke_cluster_to_project ADD CONSTRAINT l_gcp_gke_cluster_to_project_cluster_id_fkey FOREIGN KEY (cluster_id) REFERENCES gcp_gke_cluster (id) ON DELETE CASCADE;
ALTER TABLE l_gcp_gke_cluster_to_project ADD CONSTRAINT l_gcp_gke_cluster_to_project_project_id_fkey FOREIGN KEY (project_id) REFERENCES gcp_project (id) ON DELETE CASCADE;
ALTER TABLE l_az_rg_to_subscription ADD CONSTRAINT l_az_rg_to_subscription_rg_id_fkey FOREIGN KEY (rg_id) REFERENCES az_resource_group (id) ON DELETE CASCADE;
ALTER TABLE l_az_rg_to_subscription ADD CONSTRAINT l_az_rg_to_subscription_sub_id_fkey FOREIGN KEY (sub_id) REFERENCES az_subscription (id) ON DELETE CASCADE;
ALTER TABLE l_az_vm_to_rg ADD CONSTRAINT l_az_vm_to_rg_rg_id_fkey FOREIGN KEY (rg_id) REFERENCES az_resource_group (id) ON DELETE CASCADE;
ALTER TABLE l_az_vm_to_rg ADD CONSTRAINT l_az_vm_to_rg_vm_id_fkey FOREIGN KEY (vm_id) REFERENCES az_vm (id) ON DELETE CASCADE;
ALTER TABLE l_az_pub_addr_to_rg ADD CONSTRAINT l_az_pub_addr_to_rg_rg_id_fkey FOREIGN KEY (rg_id) REFERENCES az_resource_group (id) ON DELETE CASCADE;
ALTER TABLE l_az_pub_addr_to_rg ADD CONSTRAINT l_az_pub_addr_to_rg_pa_id_fkey FOREIGN KEY (pa_id) REFERENCES az_public_address (id) ON DELETE CASCADE;
ALTER TABLE l_az_lb_to_rg ADD CONSTRAINT l_az_lb_to_rg_rg_id_fkey FOREIGN KEY (rg_id) REFERENCES az_resource_group (id) ON DELETE CASCADE;
ALTER TABLE l_az_lb_to_rg ADD CONSTRAINT l_az_lb_to_rg_lb_id_fkey FOREIGN KEY (lb_id) REFERENCES az_lb (id) ON DELETE CASCADE;
ALTER TABLE l_az_vpc_to_rg ADD CONSTRAINT l_az_vpc_to_rg_rg_id_fkey FOREIGN KEY (rg_id) REFERENCES az_resource_group (id) ON DELETE CASCADE;
ALTER TABLE l_az_vpc_to_rg ADD CONSTRAINT l_az_vpc_to_rg_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES az_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_az_subnet_to_vpc ADD CONSTRAINT l_az_subnet_to_vpc_subnet_id_fkey FOREIGN KEY (subnet_id) REFERENCES az_subnet (id) ON DELETE CASCADE;
ALTER TABLE l_az_subnet_to_vpc ADD CONSTRAINT l_az_subnet_to_vpc_vpc_id_fkey FOREIGN KEY (vpc_id) REFERENCES az_vpc (id) ON DELETE CASCADE;
ALTER TABLE l_az_blob_container_to_rg ADD CONSTRAINT l_az_blob_container_to_rg_blob_container_id_fkey FOREIGN KEY (blob_container_id) REFERENCES az_blob_container (id) ON DELETE CASCADE;
ALTER TABLE l_az_blob_container_to_rg ADD CONSTRAINT l_az_blob_container_to_rg_rg_id_fkey FOREIGN KEY (rg_id) REFERENCES az_resource_group (id) ON DELETE CASCADE;

--
-- Recreate the views
--
CREATE OR REPLACE VIEW aws_loadbalancer_interface AS
 SELECT lb.id AS lb_id,
    lb.name AS lb_name,
    lb.dns_name,
    lb.vpc_id,
    lb.region_name,
    lb.type AS lb_type,
    lb.account_id,
    ni.id AS ni_id,
    ni.subnet_id,
    ni.interface_type,
    ni.mac_address,
    ni.private_ip_address,
    ni.public_ip_address
   FROM aws_loadbalancer lb
     JOIN l_aws_lb_to_net_interface link ON lb.id = link.lb_id
     JOIN aws_net_interface ni ON ni.id = link.ni_id;

CREATE OR REPLACE VIEW aws_orphan_bucket AS
 SELECT b.creation_date,
    b.region_name,
    b.id,
    b.created_at,
    b.updated_at,
    b.account_id
   FROM aws_bucket b
     LEFT JOIN g_backup_bucket gbb ON b.name::text = gbb.name::text
  WHERE gbb.name IS NULL;

CREATE OR REPLACE VIEW aws_orphan_instance AS
 SELECT i.name,
    i.arch,
    i.instance_id,
    i.instance_type,
    i.state,
    i.subnet_id,
    i.vpc_id,
    i.platform,
    i.id,
    i.created_at,
    i.updated_at,
    i.region_name,
    i.image_id,
    i.launch_time,
    i.account_id,
    v.name AS vpc_name,
    s.name AS shoot_name,
    s.project_name,
    s.technical_id AS shoot_technical_id
   FROM aws_instance i
     LEFT JOIN g_machine m ON i.name::text = m.name::text
     LEFT JOIN aws_vpc v ON i.vpc_id::text = v.vpc_id::text AND i.account_id::text = v.account_id::text
     LEFT JOIN g_shoot s ON v.name::text = s.technical_id::text
  WHERE m.name IS NULL;

CREATE OR REPLACE VIEW aws_unknown_instance_image AS
 SELECT DISTINCT i.name,
    i.arch,
    i.instance_id,
    i.instance_type,
    i.state,
    i.subnet_id,
    i.vpc_id,
    i.platform,
    i.id,
    i.created_at,
    i.updated_at,
    i.region_name,
    i.image_id,
    i.launch_time,
    i.account_id,
    s.name AS shoot_name,
    s.technical_id AS shoot_technical_id,
    s.project_name
   FROM aws_instance i
     JOIN g_machine m ON i.name::text = m.name::text
     JOIN g_shoot s ON m.namespace::text = s.technical_id::text
     LEFT JOIN g_cloud_profile_aws_image cpaw ON s.cloud_profile::text = cpaw.cloud_profile_name::text AND i.image_id::text = cpaw.ami::text
  WHERE cpaw.ami IS NULL;


CREATE OR REPLACE VIEW aws_instance_interface AS
 SELECT i.name,
    i.arch,
    i.instance_id,
    i.instance_type,
    i.state,
    i.subnet_id,
    i.vpc_id,
    i.platform,
    i.id,
    i.created_at,
    i.updated_at,
    i.region_name,
    i.image_id,
    i.launch_time,
    i.account_id,
    ni.id AS net_interface_id,
    ni.private_ip_address,
    ni.public_ip_address,
    ni.mac_address
   FROM aws_instance i
     JOIN aws_net_interface ni ON i.instance_id::text = ni.instance_id::text AND i.account_id::text = ni.account_id::text;

CREATE OR REPLACE VIEW aws_orphan_vpc AS
 SELECT v.name,
    v.vpc_id,
    v.state,
    v.ipv4_cidr,
    v.ipv6_cidr,
    v.is_default,
    v.owner_id,
    v.region_name,
    v.id,
    v.created_at,
    v.updated_at,
    v.account_id
   FROM aws_vpc v
     LEFT JOIN g_shoot s ON v.name::text = s.technical_id::text
  WHERE s.technical_id IS NULL;

CREATE OR REPLACE VIEW aws_orphan_subnet AS
 SELECT s.subnet_id,
    s.vpc_id,
    s.az,
    s.subnet_arn,
    s.account_id,
    s.created_at,
    s.updated_at
   FROM aws_subnet s
     JOIN aws_orphan_vpc aov ON s.vpc_id::text = aov.vpc_id::text AND s.account_id::text = aov.account_id::text;

CREATE OR REPLACE VIEW gcp_boot_disk AS
 SELECT d.id,
    d.name,
    d.project_id,
    d.zone,
    d.region,
    d.creation_timestamp,
    d.type,
    d.description,
    d.created_at,
    d.updated_at
   FROM gcp_disk d
     JOIN gcp_instance i ON d.name::text = i.name::text AND d.project_id::text = i.project_id::text AND d.zone::text = i.zone::text;

CREATE OR REPLACE VIEW gcp_data_disk AS
 SELECT d.id,
    d.name,
    d.project_id,
    d.zone,
    d.type,
    d.region,
    d.description,
    d.is_regional,
    d.creation_timestamp,
    d.last_attach_timestamp,
    d.last_detach_timestamp,
    d.status,
    d.size_gb,
    d.k8s_cluster_name,
    d.created_at,
    d.updated_at,
    i.name AS instance_name,
    i.id AS instance_id
   FROM gcp_disk d
     JOIN gcp_instance i ON d.name::text ~~ concat(i.name, '-%') AND d.project_id::text = i.project_id::text AND d.zone::text = i.zone::text;

CREATE OR REPLACE VIEW gcp_orphan_disk AS
 SELECT d.id,
    d.name,
    d.project_id,
    d.zone,
    d.type,
    d.region,
    d.description,
    d.is_regional,
    d.creation_timestamp,
    d.last_attach_timestamp,
    d.last_detach_timestamp,
    d.status,
    d.size_gb,
    d.k8s_cluster_name,
    d.created_at,
    d.updated_at,
    gad.instance_name,
    s.is_hibernated AS shoot_is_hibernated
   FROM gcp_disk d
     LEFT JOIN g_persistent_volume gpv ON d.name::text = gpv.name::text
     LEFT JOIN g_shoot s ON d.k8s_cluster_name::text = s.technical_id::text
     LEFT JOIN gcp_attached_disk gad ON gad.disk_name::text = d.name::text
  WHERE gpv.id IS NULL AND NOT (d.id IN ( SELECT gcp_boot_disk.id
           FROM gcp_boot_disk
        UNION
         SELECT gcp_data_disk.id
           FROM gcp_data_disk));

CREATE OR REPLACE VIEW gcp_zonal_disk AS
 SELECT id,
    name,
    project_id,
    zone,
    region,
    creation_timestamp,
    type,
    description,
    created_at,
    updated_at
   FROM gcp_disk d
  WHERE is_regional = false;


CREATE OR REPLACE VIEW gcp_regional_disk AS
 SELECT id,
    name,
    project_id,
    region,
    creation_timestamp,
    type,
    description,
    created_at,
    updated_at
   FROM gcp_disk d
  WHERE is_regional = true;

CREATE OR REPLACE VIEW gcp_orphan_instance AS
 SELECT i.id,
    i.name,
    i.hostname,
    i.instance_id,
    i.project_id,
    i.region,
    i.zone,
    i.cpu_platform,
    i.status,
    i.status_message,
    i.creation_timestamp,
    i.description
   FROM gcp_instance i
     LEFT JOIN g_machine m ON i.name::text = m.name::text
  WHERE m.name IS NULL;


CREATE OR REPLACE VIEW gcp_orphan_vpc AS
 SELECT v.id,
    v.name,
    v.project_id,
    v.vpc_id,
    v.description,
    v.creation_timestamp
   FROM gcp_vpc v
     LEFT JOIN g_shoot s ON v.name::text = s.technical_id::text
  WHERE s.technical_id IS NULL;

CREATE OR REPLACE VIEW gcp_orphan_subnet AS
 SELECT s.name,
    s.region,
    s.project_id,
    s.vpc_name,
    s.creation_timestamp,
    s.created_at,
    s.updated_at
   FROM gcp_subnet s
     JOIN gcp_orphan_vpc gov ON s.vpc_name::text = gov.name::text AND s.project_id::text = gov.project_id::text;
