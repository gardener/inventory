CREATE TABLE IF NOT EXISTS "aws_loadbalancer" (
    "arn" VARCHAR NOT NULL,
    "name" VARCHAR NOT NULL,
    "dns_name" VARCHAR NOT NULL,
    "ip_address_type" VARCHAR NOT NULL,
    "canonical_hosted_zone_id" VARCHAR NOT NULL,
    "state" VARCHAR NOT NULL,
    "scheme" VARCHAR NOT NULL,
    "vpc_id" VARCHAR NOT NULL,
    "region_name" VARCHAR NOT NULL,

    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    UNIQUE ("arn")
);
