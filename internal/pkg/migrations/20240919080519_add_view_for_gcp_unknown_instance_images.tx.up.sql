CREATE OR REPLACE VIEW "gcp_unknown_instance_image" AS
SELECT
        i.instance_id,
        i.name,
        i.project_id,
        i.creation_timestamp,
        i.created_at,
        i.updated_at,
        i.source_machine_image,
        s.name AS shoot_name,
        s.technical_id as shoot_technical_id,
        s.project_name as shoot_project_name,
        cpgi.image
FROM gcp_instance AS i
INNER JOIN g_machine AS m ON i.name = m.name
INNER JOIN g_shoot AS s ON m.namespace = s.technical_id
LEFT JOIN g_cloud_profile_gcp_image AS cpgi ON s.cloud_profile = cpgi.cloud_profile_name
AND i.source_machine_image = cpgi.image
WHERE cpgi.image IS NULL;
