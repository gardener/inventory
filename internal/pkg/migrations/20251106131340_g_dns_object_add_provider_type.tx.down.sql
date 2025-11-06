-- dependent on g_dns_object. Cannot remove g_dns_object before removing this.
DROP VIEW IF EXISTS "aws_orphan_dns_record";
DROP VIEW IF EXISTS "g_dns_object";

CREATE VIEW "g_dns_object" AS 
SELECT 
    fqdn,
    name,
    namespace,
    seed_name,
    dns_zone,
    value
FROM g_dns_record
UNION
SELECT 
    fqdn,
    name,
    namespace,
    seed_name,
    dns_zone,
    value
FROM g_dns_entry;

-- pre-migration state of the view
CREATE VIEW "aws_orphan_dns_record" AS 
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
