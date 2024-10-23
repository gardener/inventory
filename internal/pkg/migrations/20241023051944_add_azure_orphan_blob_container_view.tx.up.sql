CREATE OR REPLACE VIEW "az_orphan_blob_container" AS
SELECT
        b.name,
        b.subscription_id,
        b.resource_group,
        b.storage_account,
        b.public_access,
        b.deleted,
        b.last_modified_time,
        b.created_at,
        b.updated_at
FROM az_blob_container AS b
LEFT JOIN g_backup_bucket AS gbb ON b.name = gbb.name
WHERE gbb.name IS NULL;
