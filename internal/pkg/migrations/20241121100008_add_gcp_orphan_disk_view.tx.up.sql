DROP VIEW IF EXISTS gcp_leaked_disk;
CREATE OR REPLACE VIEW "gcp_orphan_disk" AS
SELECT
    d.id,
    d.name,
    d.project_id,
    d.zone,
    d.type,
    d.region,
    d.description,
    d.is_regional,
    d.creation_timestamp,
    d.last_attach_timestamp,
    d.last_detach_timestamp,
    d.status,
    d.size_gb,
    d.k8s_cluster_name,
    d.created_at,
    d.updated_at,
    gad.instance_name,
    s.is_hibernated AS shoot_is_hibernated
FROM gcp_disk AS d
LEFT JOIN g_persistent_volume as gpv ON gpv.disk_ref LIKE concat('%', d.name)
LEFT JOIN g_shoot AS s ON d.k8s_cluster_name = s.technical_id
LEFT JOIN gcp_attached_disk AS gad ON gad.disk_name = d.name
WHERE
    gpv.id IS NULL
    AND d.id NOT IN (SELECT id FROM gcp_boot_disk UNION SELECT id FROM gcp_data_disk);
