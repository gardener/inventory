CREATE OR REPLACE VIEW "aws_orphan_dhcp_option_set" AS
SELECT
    d.set_id,
    d.name,
    d.account_id,
    d.region_name
FROM aws_dhcp_option_set as d
LEFT JOIN aws_vpc as v ON d.set_id = v.dhcp_option_set_id
WHERE  v.vpc_id IS NULL;
