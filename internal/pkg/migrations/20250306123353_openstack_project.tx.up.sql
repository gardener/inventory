CREATE TABLE IF NOT EXISTS "openstack_project" (
    "project_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "region" varchar NOT NULL,
    "parent_id" varchar NOT NULL,
    "description" varchar NOT NULL,
    "enabled" boolean NOT NULL,
    "is_domain" boolean NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_project_key" UNIQUE ("project_id")
);
