CREATE OR REPLACE VIEW "openstack_orphan_subnet" AS 
SELECT
    s.subnet_id,
    s.name as subnet_name,
    s.network_id,
    n.name as network_name,
    p.project_id as project_id,
    p.name as project_name
FROM openstack_subnet as s
JOIN openstack_orphan_network as n ON s.network_id = n.network_id
JOIN openstack_project as p ON s.project_id = p.project_id;
