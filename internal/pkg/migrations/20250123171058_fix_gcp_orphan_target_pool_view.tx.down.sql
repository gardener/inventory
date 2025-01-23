DROP VIEW IF EXISTS "gcp_orphan_public_address";
DROP VIEW IF EXISTS "gcp_orphan_target_pool";

--
-- The `gcp_orphan_target_pool' view returns the list of GCP Target Pools, which
-- have been identified to have at least one Target Pool Instance without a
-- backing GCE Virtual Machine.
--
CREATE VIEW "gcp_orphan_target_pool" AS
SELECT DISTINCT
       tp.id,
       tp.name,
       tp.project_id,
       tp.description,
       tp.target_pool_id,
       tp.backup_pool,
       tp.creation_timestamp,
       tp.region,
       tp.security_policy,
       tp.session_affinity,
       tp.created_at,
       tp.updated_at
FROM gcp_target_pool AS tp
INNER JOIN gcp_orphan_target_pool_instance AS otpi ON tp.id = otpi.id;

--
-- The `gcp_orphan_public_address' view returns the list of GCP Load Balancer
-- Frontends (Forwarding Rules), which are associated with a Target Pool
-- Backend, that contains at least one Target Pool Instance without the
-- corresponding GCE Virtual Machine.
CREATE OR REPLACE VIEW "gcp_orphan_public_address" AS
SELECT
        fr.rule_id,
        fr.project_id,
        fr.name,
        fr.ip_address,
        fr.ip_protocol,
        fr.ip_version,
        fr.all_ports,
        fr.allow_global_access,
        fr.backend_service,
        fr.base_forwarding_rule,
        fr.creation_timestamp,
        fr.description,
        fr.load_balancing_scheme,
        fr.network,
        fr.network_tier,
        fr.port_range,
        fr.ports,
        fr.region,
        fr.service_label,
        fr.service_name,
        fr.source_ip_ranges,
        fr.subnetwork,
        fr.target,
        fr.created_at,
        fr.updated_at,
        fr.id
FROM gcp_forwarding_rule AS fr
INNER JOIN gcp_orphan_target_pool AS otp ON fr.project_id = otp.project_id AND fr.name = otp.name
WHERE fr.load_balancing_scheme = 'EXTERNAL';
