CREATE OR REPLACE VIEW aws_orphan_vpc AS
SELECT
        v.*
FROM aws_vpc AS v
LEFT JOIN g_shoot AS s ON v.name = s.technical_id
WHERE s.technical_id is NULL;
