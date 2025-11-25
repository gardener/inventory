CREATE OR REPLACE VIEW "gcp_orphan_instance" AS
SELECT
    i.id,
    i.name,
    i.hostname,
    i.instance_id,
    i.project_id,
    i.region,
    i.zone,
    i.cpu_platform,
    i.status,
    i.status_message,
    i.creation_timestamp,
    i.description,
    i.last_start_timestamp,
    i.last_stop_timestamp,
    i.last_suspend_timestamp,
    i.machine_type,
    i.gke_cluster_name,
    i.gke_pool_name
FROM gcp_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name
LEFT JOIN gcp_bastion_instance as bi ON i.instance_id = bi.instance_id
WHERE i.status = 'RUNNING' AND m.name IS NULL AND bi.instance_id IS NULL;
