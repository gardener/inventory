CREATE OR REPLACE VIEW "gcp_regional_disk" AS 
SELECT
    d.id,
    d.name,
    d.project_id,
    d.region,
    d.creation_timestamp,
    d.type,
    d.description,
    d.created_at,
    d.updated_at
FROM gcp_disk AS d
WHERE d.is_regional = TRUE;
