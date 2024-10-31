CREATE OR REPLACE VIEW "az_unknown_image" AS
SELECT
        vm.name as vm_name,
        vm.subscription_id,
        vm.resource_group,
        vm.location,
        vm.power_state,
        vm.created_at,
        vm.updated_at,
        s.name AS shoot_name,
        s.technical_id as shoot_technical_id,
        s.project_name as shoot_project_name,
        cpai.name image_name,
        vm.gallery_image_id
FROM az_vm AS vm
INNER JOIN g_machine AS m ON vm.name = m.name
INNER JOIN g_shoot AS s ON m.namespace = s.technical_id
LEFT JOIN g_cloud_profile_azure_image AS cpai ON s.cloud_profile = cpai.cloud_profile_name
AND vm.gallery_image_id = cpai.gallery_image_id
WHERE cpai.name IS NULL;

