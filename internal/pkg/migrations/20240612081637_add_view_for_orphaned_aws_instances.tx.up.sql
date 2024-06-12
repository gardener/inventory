CREATE OR REPLACE VIEW aws_orphan_instance AS
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
