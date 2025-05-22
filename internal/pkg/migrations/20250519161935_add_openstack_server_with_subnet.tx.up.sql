CREATE OR REPLACE VIEW "openstack_server_with_subnet" AS
SELECT 
    s.id as server_pk,
    s.server_id,
    s.name as server_name,
    s.project_id,
    s.domain,
    s.region,
    s.availability_zone,
    s.status,
    s.image_id,
    s.server_created_at,
    s.server_updated_at,
    subnet.id as subnet_pk,
    subnet.subnet_id,
    subnet.name as subnet_name,
    subnet.network_id,
    subnet.gateway_ip,
    subnet.subnet_pool_id,
    subnet.enable_dhcp,
    subnet.ip_version
FROM openstack_server as s
INNER JOIN openstack_port AS p ON s.server_id = p.device_id
INNER JOIN openstack_port_ip AS pip ON p.port_id = pip.port_id
INNER JOIN openstack_subnet AS subnet ON pip.subnet_id = subnet.subnet_id;
