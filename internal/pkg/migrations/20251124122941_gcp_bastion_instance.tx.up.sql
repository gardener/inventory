CREATE OR REPLACE VIEW "gcp_bastion_instance" AS
SELECT
    i.name AS instance_name,
    i.instance_id,
    i.project_id AS instance_project_id,
    i.region AS instance_region,
    i.creation_timestamp AS instance_creation_timestamp,
    i.status AS instance_status,
    b.name AS bastion_name,
    b.namespace AS bastion_namespace,
    b.seed_name AS bastion_seed,
    b.ip AS bastion_ip
FROM gcp_instance AS i
JOIN gcp_nic AS nic ON i.project_id = nic.project_id AND i.instance_id = nic.instance_id
JOIN g_bastion AS b on nic.nat_ip = b.ip;
