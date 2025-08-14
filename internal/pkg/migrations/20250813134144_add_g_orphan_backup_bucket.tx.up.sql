CREATE OR REPLACE VIEW "g_orphan_backup_bucket" AS 
SELECT 
    bb.name,
    bb.provider_type,
    bb.region_name,
    bb.seed_name,
    bb.created_at,
    bb.updated_at,
    bb.state,
    bb.state_progress,
    bb.creation_timestamp
FROM g_backup_bucket as bb
LEFT JOIN g_seed as s on bb.seed_name = s.name
WHERE s.id IS NULL;
