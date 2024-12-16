--
-- The `gcp_orphan_target_pool_instance' view returns the list of GCP Target
-- Pool instances, which do not have a corresponding GCE Virtual Machine.
--
-- If the instance is mapped to a shoot, in order to consider the target pool
-- instance as orphan the shoot should either be running or not existing at all.
CREATE OR REPLACE VIEW "gcp_orphan_target_pool_instance" AS
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
        tp.updated_at,
        tpi.instance_name
FROM gcp_target_pool_instance AS tpi
INNER JOIN gcp_target_pool AS tp ON tpi.project_id = tp.project_id AND tpi.target_pool_id = tp.target_pool_id
LEFT JOIN gcp_instance AS i ON tpi.project_id = i.project_id AND tpi.instance_name = i.name
LEFT JOIN g_shoot AS gs ON tpi.inferred_g_shoot = gs.technical_id
WHERE i.name IS NULL AND (gs.is_hibernated IS NULL OR gs.is_hibernated = false);
