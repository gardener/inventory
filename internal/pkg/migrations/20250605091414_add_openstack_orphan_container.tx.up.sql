CREATE OR REPLACE VIEW "openstack_orphan_container" AS
SELECT
    c.name,
    c.project_id,
    c.bytes,
    c.object_count,
    c.created_at,
    c.updated_at
FROM openstack_container AS c
LEFT JOIN g_backup_bucket gbb ON c.name = gbb.name AND gbb.provider_type = 'openstack'
WHERE gbb.name IS NULL;
