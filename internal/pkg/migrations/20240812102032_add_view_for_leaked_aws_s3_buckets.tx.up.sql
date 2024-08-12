CREATE OR REPLACE VIEW "aws_orphan_bucket"
AS
SELECT
        b.*
FROM aws_bucket AS b
LEFT JOIN g_backup_bucket AS gbb ON b.name = gbb.name
WHERE gbb.name IS NULL;
