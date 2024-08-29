CREATE TABLE IF NOT EXISTS "gcp_vpc" (
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "vpc_id" bigint NOT NULL,
    "creation_timestamp" varchar,
    "description" varchar NOT NULL,
    "gateway_ipv4" varchar NOT NULL,
    "firewall_policy" varchar NOT NULL,
    "mtu" varchar NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_vpc_key" UNIQUE ("vpc_id", "project_id")
);
