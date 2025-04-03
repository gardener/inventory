CREATE OR REPLACE VIEW "openstack_orphan_loadbalancer" AS
SELECT
    lb.id,
    lb.name,
    lb.status,
    lb.provider,
    lb.vip_address,
    lb.vip_network_id,
    lb.vip_subnet_id,
    lb.loadbalancer_created_at,
    lb.loadbalancer_updated_at,
    lb.project_id
FROM openstack_loadbalancer as lb
LEFT JOIN openstack_orphan_network as n
ON lb.vip_network_id = n.network_id
WHERE n.network_id IS NULL;
