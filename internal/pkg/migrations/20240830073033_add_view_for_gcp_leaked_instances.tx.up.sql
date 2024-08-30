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
        i.description
FROM gcp_instance AS i
LEFT JOIN g_machine AS m ON i.name = m.name
WHERE m.name IS NULL;
