CREATE OR REPLACE VIEW "gcp_boot_disk" AS
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
INNER JOIN gcp_instance as i 
ON d.name = i.name AND d.project_id = i.project_id and d.zone = i.zone;
