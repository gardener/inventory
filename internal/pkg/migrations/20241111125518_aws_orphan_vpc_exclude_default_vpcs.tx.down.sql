CREATE OR REPLACE VIEW "aws_orphan_vpc" AS
SELECT 
    v.name,
    v.vpc_id,
    v.state,
    v.ipv4_cidr,
    v.region_name,
    v.is_default,
    v.owner_id,
    v.account_id,
    v.created_at,
    v.updated_at,
FROM aws_vpc AS v
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE s.technical_id IS NULL;
