CREATE OR REPLACE VIEW "gcp_public_address" AS
SELECT
        ga.address AS ip_address,
        ga.region AS region,
        ga.project_id AS project_id,
        'gcp_address' AS origin
FROM gcp_address AS ga WHERE ga.address_type = 'EXTERNAL'
UNION
SELECT
        gfr.ip_address AS ip_address,
        gfr.region AS region,
        gfr.project_id AS project_id,
        'gcp_forwarding_rule' AS origin
FROM gcp_forwarding_rule AS gfr WHERE gfr.load_balancing_scheme = 'EXTERNAL';
