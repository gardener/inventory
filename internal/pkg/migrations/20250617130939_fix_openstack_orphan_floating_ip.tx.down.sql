DROP VIEW IF EXISTs "openstack_orphan_floating_ip";

CREATE OR REPLACE VIEW "openstack_orphan_floating_ip" AS
SELECT
    ip.id,
    ip.fixed_ip,
    ip.floating_ip,
    ip.floating_network_id,
    ip.ip_created_at,
    ip.ip_updated_at,
    ip.project_id
FROM openstack_floating_ip as ip
JOIN openstack_orphan_network as n
ON ip.floating_network_id = n.network_id;
