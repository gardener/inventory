CREATE TABLE IF NOT EXISTS "openstack_loadbalancer" (
    "loadbalancer_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "region" varchar NOT NULL,
    "status" varchar NOT NULL,
    "provider" varchar NOT NULL,
    "vip_address" varchar NOT NULL,
    "vip_network_id" varchar NOT NULL,
    "vip_subnet_id" varchar NOT NULL,
    "description" varchar NOT NULL,
    "loadbalancer_created_at" timestamptz NOT NULL,
    "loadbalancer_updated_at" timestamptz NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_loadbalancer_key" UNIQUE ("loadbalancer_id", "project_id")
);
