CREATE OR REPLACE VIEW "gcp_data_disk" AS
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
    i.name AS instance_name,
    i.id AS instance_id
FROM gcp_disk AS d
INNER JOIN gcp_instance as i
ON d.name LIKE concat(i.name, '-%') AND d.project_id = i.project_id and d.zone = i.zone;
