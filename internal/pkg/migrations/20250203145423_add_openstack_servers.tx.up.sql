CREATE TABLE IF NOT EXISTS "openstack_server" (
    "server_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "region" varchar NOT NULL,
    "user_id" varchar,
    "availability_zone" varchar,
    "status" varchar,
    "image_id" varchar,
    "server_created_at" timestamptz,
    "server_updated_at" timestamptz,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_server_key" UNIQUE ("server_id", "project_id")
);
