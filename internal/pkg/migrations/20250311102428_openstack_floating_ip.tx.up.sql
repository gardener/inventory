CREATE TABLE IF NOT EXISTS "openstack_floating_ip" (
    "floating_ip_id" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "region" varchar NOT NULL,
    "port_id" varchar NOT NULL,
    "fixed_ip" inet NOT NULL,
    "router_id" varchar NOT NULL,
    "floating_ip" inet NOT NULL,
    "floating_network_id" varchar NOT NULL,
    "description" varchar NOT NULL,
    "ip_created_at" timestamptz NOT NULL,
    "ip_updated_at" timestamptz NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_floating_ip_key" UNIQUE ("floating_ip_id", "project_id")
);
