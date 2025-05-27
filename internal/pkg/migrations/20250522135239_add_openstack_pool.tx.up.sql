CREATE TABLE IF NOT EXISTS "openstack_pool" (
    "pool_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "subnet_id" varchar NOT NULL,
    "description" varchar NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_pool_key" UNIQUE ("pool_id", "project_id")
);
