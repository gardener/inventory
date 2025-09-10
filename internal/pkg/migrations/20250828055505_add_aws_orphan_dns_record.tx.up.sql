CREATE OR REPLACE VIEW "aws_orphan_dns_record" AS 
SELECT
    adr.name,
    adr.type,
    adr.value,
    adr.account_id,
    adr.hosted_zone_id
FROM aws_dns_record as adr
LEFT JOIN g_dns_object as gdo
ON adr.name = gdo.fqdn || '.'
AND adr.type NOT IN ('NS', 'SOA', 'MX', 'PTR')
WHERE gdo.name IS NULL;
