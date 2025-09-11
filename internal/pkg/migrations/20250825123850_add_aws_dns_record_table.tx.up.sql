CREATE TABLE IF NOT EXISTS "aws_dns_record" (
    "account_id" TEXT NOT NULL,
    "hosted_zone_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "set_identifier" TEXT,
    "is_alias" BOOLEAN NOT NULL,
    "ttl" BIGINT,
    "evaluate_health" BOOLEAN,
    "value" TEXT,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "aws_record_key" UNIQUE ("account_id", "hosted_zone_id", "name", "type", "set_identifier", "value")
);
