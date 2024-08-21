--
-- Load Balancers
--
DROP VIEW IF EXISTS "aws_loadbalancer_interface";
CREATE OR REPLACE VIEW "aws_loadbalancer_interface" AS
SELECT
        lb.id AS lb_id,
        lb.name AS lb_name,
        lb.dns_name AS dns_name,
        lb.vpc_id AS vpc_id,
        lb.region_name AS region_name,
        lb.type AS lb_type,
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
-- S3 buckets
--
DROP VIEW IF EXISTS "aws_orphan_bucket";
CREATE OR REPLACE VIEW "aws_orphan_bucket"
AS
SELECT
        b.creation_date,
        b.region_name,
        b.id,
        b.created_at,
        b.updated_at
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
        v.region_name AS region,
        s.name AS shoot_name,
        s.project_name AS project_name
FROM aws_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name
LEFT JOIN aws_vpc AS v ON i.vpc_id = v.vpc_id
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
DROP VIEW IF EXISTS "aws_unknown_instance_image";
--
-- Instance Interfaces
--
DROP VIEW IF EXISTS "aws_instance_interface";
