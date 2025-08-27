CREATE TABLE IF NOT EXISTS "aws_record_set" (
    "region_name" TEXT NOT NULL,
    "account_id" TEXT NOT NULL,
    "hosted_zone_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "set_identifier" TEXT,
    "is_alias" BOOLEAN NOT NULL,
    "ttl" BIGINT,
    "alias_dns_name" TEXT,
    "evaluate_health" BOOLEAN,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "aws_record_set_key" UNIQUE ("hosted_zone_id", "account_id", "name", "type", "set_identifier")
);
