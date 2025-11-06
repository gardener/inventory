CREATE OR REPLACE VIEW "g_dns_object" AS 
SELECT 
    fqdn,
    name,
    namespace,
    seed_name,
    dns_zone,
    value,
    provider_type
FROM g_dns_record
UNION
SELECT 
    fqdn,
    name,
    namespace,
    seed_name,
    dns_zone,
    value,
    provider_type
FROM g_dns_entry;
