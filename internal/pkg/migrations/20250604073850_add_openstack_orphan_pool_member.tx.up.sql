CREATE OR REPLACE VIEW "openstack_orphan_pool_member" AS
SELECT
    p.pool_id,
    p.name as pool_name,
    p.project_id,
    pm.member_id,
    pm.name,
    pm.inferred_gardener_shoot,
    pm.protocol_port,
    pm.member_created_at,
    pm.member_updated_at
FROM openstack_pool_member AS pm
INNER JOIN openstack_pool AS p ON pm.project_id = p.project_id AND pm.pool_id = p.pool_id
LEFT JOIN openstack_server AS s ON pm.project_id = s.project_id AND pm.name = s.name
LEFT JOIN g_shoot AS gs ON pm.inferred_gardener_shoot = gs.technical_id
WHERE s.name IS NULL AND (gs.is_hibernated IS NULL OR gs.is_hibernated = false);

-- INNER JOIN openstack_pool AS p ON pm.project_id = p.project_id AND pm.pool_id = p.pool_id
-- LEFT JOIN openstack_server AS s ON pm.project_id = s.project_id AND pm.name = s.name
-- WHERE s.name IS NULL;
