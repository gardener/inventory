CREATE OR REPLACE VIEW "az_bastion_vm" AS
SELECT
    vm.name as vm_name,
    vm.subscription_id,
    vm.resource_group,
    vm.location,
    vm.ip_address,
    vm.vm_created_at,
    b.name as bastion_name,
    b.namespace as bastion_namespace,
    b.seed_name as bastion_seed
FROM az_vm_public_address as vm
JOIN g_bastion as b
ON vm.ip_address = b.ip;
