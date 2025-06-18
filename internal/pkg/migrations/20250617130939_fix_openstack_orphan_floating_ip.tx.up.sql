DROP VIEW IF EXISTS "openstack_orphan_floating_ip";

CREATE OR REPLACE VIEW "openstack_orphan_floating_ip" AS
SELECT
    fip.floating_ip_id,
    fip.project_id AS ip_project_id,
    fip.domain AS ip_domain,
    fip.region AS ip_region,
    fip.port_id,
    fip.router_id,
    fip.project_id,
    p.device_id,
    lb.loadbalancer_id,
    lb.name AS loadbalancer_name
FROM openstack_floating_ip AS fip
JOIN openstack_port AS p ON fip.port_id = p.port_id AND fip.project_id = p.project_id
LEFT JOIN openstack_loadbalancer AS lb ON p.device_id = lb.loadbalancer_id
WHERE lb.id IS NULL OR lb.loadbalancer_id IN (SELECT olb.loadbalancer_id FROM openstack_orphan_loadbalancer olb);
