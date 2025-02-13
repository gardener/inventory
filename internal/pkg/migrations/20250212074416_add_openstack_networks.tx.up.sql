CREATE TABLE IF NOT EXISTS "openstack_network" (
    "network_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "region" varchar NOT NULL,
    "status" varchar NOT NULL,
    "description" varchar,
    "network_created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "network_updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("network_id")
);
