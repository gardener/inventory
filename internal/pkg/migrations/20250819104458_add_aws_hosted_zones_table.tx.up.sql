SET statement_timeout = 0;

CREATE TABLE "aws_hosted_zone" (
    "hosted_zone_id" TEXT NOT NULL,
    "account_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "caller_reference" TEXT NOT NULL,
    "comment" TEXT,
    "is_private" BOOLEAN NOT NULL DEFAULT FALSE,
    "resource_record_set_count" BIGINT NOT NULL DEFAULT 0,
    "region_name" TEXT NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "aws_hosted_zone_key" UNIQUE ("hosted_zone_id", "account_id")
);
