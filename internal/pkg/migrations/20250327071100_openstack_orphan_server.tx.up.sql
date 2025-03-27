CREATE OR REPLACE VIEW "openstack_orphan_server" AS
SELECT
        s.server_id,
        s.name,
        s.project_id,
        s.domain,
        s.region,
        s.user_id,
        s.availability_zone,
        s.status,
        s.image_id,
        s.server_created_at,
        s.server_updated_at,
        s.id,
        s.created_at,
        s.updated_at,
        p.name as project_name
FROM openstack_server AS s
LEFT JOIN g_machine AS m ON s.name = m.name
INNER JOIN openstack_project as p on s.project_id = p.project_id
WHERE m.name IS NULL;
