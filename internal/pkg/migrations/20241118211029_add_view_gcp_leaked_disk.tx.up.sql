CREATE OR REPLACE VIEW "gcp_leaked_disk" AS 
SELECT
    d.id,
    d.name,
    d.project_id,
    d.zone,
    d.region,
    d.creation_timestamp,
    d.type,
    d.description,
    d.created_at,
    d.updated_at
FROM gcp_disk AS d
LEFT JOIN g_persistent_volume as pv ON pv.disk_ref LIKE '%' || d.name
WHERE pv.name IS NULL and d.id NOT IN (select bd.id from gcp_boot_disk as bd);
