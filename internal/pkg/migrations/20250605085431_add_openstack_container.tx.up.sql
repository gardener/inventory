CREATE TABLE IF NOT EXISTS "openstack_container" (
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "bytes" bigint NOT NULL,
    "object_count" bigint NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_container_key" UNIQUE ("name", "project_id")
);
