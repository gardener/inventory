CREATE OR REPLACE VIEW az_orphan_vm AS
SELECT
        vm.name,
        vm.subscription_id,
        vm.resource_group,
        vm.location,
        vm.provisioning_state,
        vm.vm_created_at,
        vm.hyper_v_gen,
        vm.vm_size,
        vm.power_state,
        vm.vm_agent_version,
        s.name AS shoot_name,
        s.project_name AS project_name
FROM az_vm AS vm
LEFT JOIN g_machine AS m ON vm.name = m.name
LEFT JOIN g_shoot AS s ON vm.resource_group = s.technical_id
WHERE m.name IS NULL;
