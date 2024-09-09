CREATE TABLE IF NOT EXISTS "gcp_subnet" (
    "subnet_id" bigint NOT NULL,
    "vpc_name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "region" varchar NOT NULL,
    "creation_timestamp" varchar,
    "description" varchar NOT NULL,
    "ipv4_cidr_range" varchar NOT NULL,
    "gateway" inet,
    "purpose" varchar NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_subnet_key" UNIQUE ("subnet_id", "vpc_name", "project_id")
);
