DROP VIEW IF EXISTS "gcp_orphan_public_address";
DROP VIEW IF EXISTS "gcp_orphan_target_pool";

--
-- The `gcp_orphan_target_pool' view returns the list of GCP Target Pools, for
-- which *all* target pool instances do not have a backing GCE instance.  If the
-- target pool instances belong to a hibernated shoot cluster they will not be
-- considered as leaked.
--
CREATE VIEW "gcp_orphan_target_pool" AS
SELECT
    tp.name,
    tp.project_id,
    tp.target_pool_id,
    COUNT(tpi.instance_name) AS num_tp_instances,
    COUNT(i.name) AS num_gce_instances
FROM gcp_target_pool AS tp
INNER JOIN gcp_target_pool_instance AS tpi ON tpi.project_id = tp.project_id AND tpi.target_pool_id = tp.target_pool_id
LEFT JOIN gcp_instance AS i ON tpi.project_id = i.project_id AND tpi.instance_name = i.name
LEFT JOIN g_shoot AS gs ON tpi.inferred_g_shoot = gs.technical_id
GROUP BY tp.name, tp.target_pool_id, tp.project_id
HAVING bool_and((i.name IS NULL) AND (gs.is_hibernated IS NULL OR gs.is_hibernated = false));

--
-- The `gcp_orphan_public_address' view returns the list of GCP Load Balancer
-- Frontends (Forwarding Rules), which are associated with a Target Pool
-- Backend, for which *all* target pool instances do not have the corresponding
-- GCE instances.
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
