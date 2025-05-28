CREATE TABLE IF NOT EXISTS "openstack_loadbalancer_with_pool" (
    "loadbalancer_id" varchar NOT NULL,
    "pool_id" varchar NOT NULL,
    "project_id" varchar NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_loadbalancer_with_pool_key" UNIQUE ("loadbalancer_id", "pool_id", "project_id")
);
