CREATE OR REPLACE VIEW "aws_orphan_instance" AS
SELECT
    i.name,
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
FROM aws_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name
LEFT JOIN aws_vpc AS v ON i.vpc_id = v.vpc_id AND i.account_id = v.account_id
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE i.state = 'running' AND m.name IS NULL;
