--
-- The `gcp_orphan_target_pool' view returns the list of GCP Target Pools, which
-- have been identified to have at least one Target Pool Instance without a
-- backing GCE Virtual Machine.
--
CREATE OR REPLACE VIEW "gcp_orphan_target_pool" AS
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
