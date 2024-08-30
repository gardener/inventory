CREATE OR REPLACE VIEW "gcp_orphan_vpc" AS
SELECT
        v.id,
        v.name,
        v.project_id,
        v.vpc_id,
        v.description,
        v.creation_timestamp
FROM gcp_vpc AS v
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE s.technical_id is NULL;
