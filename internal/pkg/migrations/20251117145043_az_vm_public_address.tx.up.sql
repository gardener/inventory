CREATE OR REPLACE VIEW "az_vm_public_address" AS 
SELECT
    vm.name,
    vm.subscription_id,
    vm.resource_group,
    vm.vm_created_at,
    addr.location,
    addr.ip_address
FROM az_public_address AS addr
JOIN az_network_interface AS nic ON 
    addr.subscription_id = nic.subscription_id AND
    addr.resource_group = nic.resource_group AND
    addr.name = nic.public_ip_name
JOIN az_vm as vm ON
    nic.subscription_id = vm.subscription_id AND
    nic.resource_group = vm.resource_group AND
    nic.vm_name = vm.name;
