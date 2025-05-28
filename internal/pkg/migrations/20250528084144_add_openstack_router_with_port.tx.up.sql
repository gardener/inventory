CREATE OR REPLACE VIEW "openstack_router_with_port" AS
SELECT
    r.router_id,
    r.name as router_name,
    r.project_id,
    r.domain,
    r.region,
    r.status as router_status,
    r.description,
    r.external_network_id,
    r.created_at,
    r.updated_at,
    p.port_id,
    pip.ip_address,
    pip.subnet_id
FROM openstack_router as r
INNER JOIN openstack_port as p ON r.router_id = p.device_id
INNER JOIN openstack_port_ip as pip on p.port_id = pip.port_id;
