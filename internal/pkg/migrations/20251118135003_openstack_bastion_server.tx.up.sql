CREATE OR REPLACE VIEW "openstack_bastion_server" AS
SELECT 
    s.server_id,
    s.name as server_name,
    s.domain as server_domain,
    s.region as server_region,
    s.project_id as server_project_id,
    s.server_created_at,
    b.name as bastion_name,
    b.namespace as bastion_namespace,
    b.seed_name as bastion_seed,
    b.ip
FROM g_bastion as b
JOIN openstack_floating_ip as fip ON b.ip = fip.floating_ip
JOIN openstack_port as p on fip.port_id = p.port_id and fip.project_id = p.project_id
JOIN openstack_server as s on p.device_id = s.server_id and p.project_id = s.project_id;
