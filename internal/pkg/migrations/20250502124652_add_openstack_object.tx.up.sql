CREATE TABLE IF NOT EXISTS "openstack_object" (
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "container_name" varchar NOT NULL,
    "content_type" varchar NOT NULL,
    "last_modified" timestamptz NOT NULL,
    "is_latest" boolean NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "openstack_object_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "openstack_object_key" UNIQUE ("name", "project_id", "container_name")
);
