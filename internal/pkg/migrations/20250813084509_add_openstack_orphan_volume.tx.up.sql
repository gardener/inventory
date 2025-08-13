CREATE OR REPLACE VIEW "openstack_orphan_volume" AS
SELECT 
    v.volume_id,
    v.name,
    v.project_id,
    v.domain,
    v.region,
    v.user_id,
    v.availability_zone,
    v.size,
    v.volume_type,
    v.status,
    v.replication_status,
    v.bootable,
    v.encrypted,
    v.multi_attach,
    v.snapshot_id,
    v.description,
    v.volume_created_at,
    v.volume_updated_at
FROM openstack_volume as v
WHERE v.availability_zone = '';
