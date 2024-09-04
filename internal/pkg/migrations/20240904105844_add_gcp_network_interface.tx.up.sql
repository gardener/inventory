CREATE TABLE IF NOT EXISTS "gcp_nic" (
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "instance_id" bigint NOT NULL,
    "network" varchar NOT NULL,
    "subnetwork" varchar NOT NULL,
    "ipv4" inet,
    "ipv6" inet,
    "ipv6_access_type" varchar NOT NULL,
    "nic_type" varchar NOT NULL,
    "stack_type" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_nic_key" UNIQUE ("name", "project_id", "instance_id")
);
