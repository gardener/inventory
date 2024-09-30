CREATE OR REPLACE VIEW "aws_orphan_subnet" AS
SELECT
        s.subnet_id,
        s.vpc_id,
        s.az,
        s.subnet_arn,
        s.account_id,
        s.created_at,
        s.updated_at
FROM aws_subnet AS s
INNER JOIN aws_orphan_vpc as aov ON s.vpc_id = aov.vpc_id and s.account_id = aov.account_id;
