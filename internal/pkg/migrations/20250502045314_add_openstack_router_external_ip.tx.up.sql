CREATE TABLE IF NOT EXISTS "openstack_router_external_ip" (
    "router_id" varchar NOT NULL,
    "external_ip" inet NOT NULL,
    "external_subnet_id" varchar NOT NULL,
    "project_id" varchar NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_router_external_ip_key" UNIQUE ("external_ip", "external_subnet_id", "router_id", "project_id")
);
