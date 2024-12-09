--
-- The `gcp_target_pool_without_frontend' view returns the list of GCP Target
-- Pools, which do not have a Forwarding Rule (frontend) configured.
--
-- It could be used to fetch GCP Load Balancers, which are left in an
-- inconsistent state.
--
CREATE OR REPLACE VIEW "gcp_target_pool_without_frontend" AS
SELECT
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
LEFT JOIN gcp_forwarding_rule AS fr ON tp.project_id = fr.project_id AND tp.name = fr.name
WHERE fr.name IS NULL;
