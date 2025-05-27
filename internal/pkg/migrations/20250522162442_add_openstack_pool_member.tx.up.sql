CREATE TABLE IF NOT EXISTS "openstack_pool_member" (
    "member_id" varchar NOT NULL,
    "pool_id" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "subnet_id" varchar NOT NULL,
    "protocol_port" varchar NOT NULL,
    "member_created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "member_updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_pool_member_key" UNIQUE ("member_id", "pool_id", "project_id")
);
