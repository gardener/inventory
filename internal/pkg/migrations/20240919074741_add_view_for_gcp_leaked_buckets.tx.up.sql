CREATE OR REPLACE VIEW "gcp_orphan_bucket" AS
SELECT
        b.name,
        b.project_id,
        b.creation_timestamp,
        b.location_type,
        b.location,
        b.created_at,
        b.updated_at
FROM gcp_bucket AS b
LEFT JOIN g_backup_bucket as gbb ON b.name = gbb.name
WHERE gbb.name IS NULL;
