CREATE TABLE IF NOT EXISTS "openstack_server" (
    "server_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "region" varchar NOT NULL,
    "user_id" varchar NOT NULL,
    "availability_zone" varchar NOT NULL,
    "status" varchar NOT NULL,
    "image_id" varchar,
    "server_created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "server_updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("server_id")
);
