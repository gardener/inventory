CREATE OR REPLACE VIEW "gcp_orphan_subnet" AS
SELECT
        s.name,
        s.region,
        s.project_id,
        s.vpc_name,
        s.creation_timestamp,
        s.created_at,
        s.updated_at
FROM gcp_subnet AS s
INNER JOIN gcp_orphan_vpc as gov ON s.vpc_name = gov.name and s.project_id = gov.project_id;
