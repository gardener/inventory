CREATE OR REPLACE VIEW "openstack_unknown_machine_image" AS
SELECT
    s.server_id,
    s.name as server_name,
    s.project_id,
    s.domain,
    s.region,
    s.user_id,
    s.availability_zone as az,
    s.status,
    s.server_created_at,
    s.server_updated_at,
    sh.name AS shoot_name,
    sh.technical_id as shoot_technical_id,
    sh.project_name as shoot_project_name,
    sh.cloud_profile
FROM openstack_server as s
INNER JOIN g_machine AS m ON s.name = m.name
INNER JOIN g_shoot AS sh ON m.namespace = sh.technical_id
LEFT JOIN g_cloud_profile_openstack_image AS cpoi ON sh.cloud_profile = cpoi.cloud_profile_name
AND s.image_id = cpoi.image_id
WHERE cpoi.name IS NULL;
