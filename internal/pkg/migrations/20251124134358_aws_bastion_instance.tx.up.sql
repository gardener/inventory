CREATE OR REPLACE VIEW "aws_bastion_instance" AS
SELECT
    b.name AS bastion_name,
    b.namespace AS bastion_namespace,
    b.seed_name AS bastion_seed,
    b.ip AS bastion_ip,
    i.instance_id,
    i.name as instance_name,
    i.state AS instance_state,
    i.account_id AS instance_account_id,
    i.region_name AS instance_region,
    i.launch_time AS instance_launch_time
FROM g_bastion as b
JOIN aws_instance_interface as i ON host(b.ip) = i.public_ip_address;
