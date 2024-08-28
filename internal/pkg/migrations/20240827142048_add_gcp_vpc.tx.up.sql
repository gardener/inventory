CREATE TABLE IF NOT EXISTS "gcp_vpc" (
    "name" varchar NOT NULL UNIQUE,
    "project_id" varchar NOT NULL,
    "vpc_id" bigint NOT NULL,
    "vpc_creation_timestamp" varchar,
    "description" varchar NOT NULL,
    "gateway_ipv4" varchar NOT NULL,
    "firewall_policy" varchar NOT NULL,
    "max_transmission_units_bytes" varchar NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_vpc_to_project_key" UNIQUE ("vpc_id", "project_id")
);
