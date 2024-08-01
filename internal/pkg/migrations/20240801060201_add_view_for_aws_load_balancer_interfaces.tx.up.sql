CREATE OR REPLACE VIEW "aws_loadbalancer_interface" AS
SELECT
        lb.id AS lb_id,
        lb.name AS lb_name,
        lb.dns_name AS dns_name,
        lb.vpc_id AS vpc_id,
        lb.region_name AS region_name,
        lb.type AS lb_type,
        ni.id AS ni_id,
        ni.subnet_id AS subnet_id,
        ni.interface_type AS interface_type,
        ni.mac_address AS mac_address,
        ni.private_ip_address AS private_ip_address,
        ni.public_ip_address AS public_ip_address
FROM aws_loadbalancer AS lb
INNER JOIN l_aws_lb_to_net_interface AS link ON lb.id = link.lb_id
INNER JOIN aws_net_interface AS ni ON ni.id = link.ni_id;
