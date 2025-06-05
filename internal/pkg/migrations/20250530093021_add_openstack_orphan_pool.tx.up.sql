CREATE OR REPLACE VIEW "openstack_orphan_pool" AS
SELECT 
    p.pool_id,
    p.name,
    p.project_id,
    COUNT(pm.name) AS num_pool_members,
    COUNT(s.name) AS num_servers
FROM "openstack_pool" AS p
    JOIN "openstack_pool_member" AS pm ON p.pool_id = pm.pool_id AND p.project_id = pm.project_id
    LEFT JOIN "openstack_server" AS s ON pm.name = s.name AND pm.project_id = s.project_id
    LEFT JOIN "g_shoot" AS gs ON pm.inferred_gardener_shoot = gs.technical_id
GROUP BY p.name, p.pool_id, p.project_id
HAVING bool_and(s.name IS NULL AND (gs.is_hibernated IS NULL or gs.is_hibernated = false));
