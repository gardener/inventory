CREATE OR REPLACE VIEW "openstack_orphan_network" AS 
SELECT 
    n.network_id,
    n.name,
    n.network_created_at,
    n.network_updated_at,
    p.name as project_name,
    p.project_id as project_id
FROM openstack_network as n
LEFT JOIN g_shoot as s ON n.name = s.technical_id
JOIN openstack_project as p on n.project_id = p.project_id
WHERE s.id IS NULL;
