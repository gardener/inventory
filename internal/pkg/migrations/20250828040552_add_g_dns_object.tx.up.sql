CREATE OR REPLACE VIEW "g_dns_object" AS
SELECT fqdn, name, namespace, seed_name, dns_zone, value
FROM g_dns_record
UNION
SELECT fqdn, name, namespace, seed_name, dns_zone, value
FROM g_dns_entry;
