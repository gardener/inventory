--
-- Load Balancers
--
DROP VIEW "aws_loadbalancer_interface";
CREATE OR REPLACE VIEW "aws_loadbalancer_interface" AS
SELECT
        lb.id AS lb_id,
        lb.name AS lb_name,
        lb.dns_name AS dns_name,
        lb.vpc_id AS vpc_id,
        lb.region_name AS region_name,
        lb.type AS lb_type,
        lb.account_id AS account_id,
        ni.id AS ni_id,
        ni.subnet_id AS subnet_id,
        ni.interface_type AS interface_type,
        ni.mac_address AS mac_address,
        ni.private_ip_address AS private_ip_address,
        ni.public_ip_address AS public_ip_address
FROM aws_loadbalancer AS lb
INNER JOIN l_aws_lb_to_net_interface AS link ON lb.id = link.lb_id
INNER JOIN aws_net_interface AS ni ON ni.id = link.ni_id;
--
-- S3 Buckets
--
DROP VIEW IF EXISTS "aws_orphan_bucket";
CREATE OR REPLACE VIEW "aws_orphan_bucket"
AS
SELECT
        b.creation_date,
        b.region_name,
        b.id,
        b.created_at,
        b.updated_at,
        b.account_id
FROM aws_bucket AS b
LEFT JOIN g_backup_bucket AS gbb ON b.name = gbb.name
WHERE gbb.name IS NULL;
--
-- EC2 Instances
--
DROP VIEW IF EXISTS "aws_orphan_instance";
CREATE OR REPLACE VIEW "aws_orphan_instance" AS
SELECT
        i.*,
        v.name AS vpc_name,
        s.name AS shoot_name,
        s.project_name AS project_name,
        s.technical_id AS shoot_technical_id
FROM aws_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name
LEFT JOIN aws_vpc AS v ON i.vpc_id = v.vpc_id AND i.account_id = v.account_id
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE m.name IS NULL;
--
-- VPCs
--
DROP VIEW IF EXISTS "aws_orphan_vpc";
CREATE OR REPLACE VIEW "aws_orphan_vpc" AS
SELECT
        v.*
FROM aws_vpc AS v
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE s.technical_id is NULL;
--
-- Unknown images
--
CREATE OR REPLACE VIEW "aws_unknown_instance_image" AS
SELECT DISTINCT
       i.*,
       s.name AS shoot_name,
       s.technical_id AS shoot_technical_id,
       s.project_name AS project_name
FROM aws_instance AS i
INNER JOIN g_machine AS m ON i.name = m.name
INNER JOIN g_shoot AS s ON m.namespace = s.technical_id
LEFT JOIN g_cloud_profile_aws_image AS cpaw ON s.cloud_profile = cpaw.cloud_profile_name AND i.image_id = cpaw.ami
WHERE cpaw.ami IS NULL;
--
-- Instance Interfaces
--
CREATE OR REPLACE VIEW "aws_instance_interface" AS
SELECT
        i.*,
        ni.id AS net_interface_id,
        ni.private_ip_address,
        ni.public_ip_address,
        ni.mac_address
FROM aws_instance AS i
INNER JOIN aws_net_interface AS ni ON i.instance_id = ni.instance_id AND i.account_id = ni.account_id;
