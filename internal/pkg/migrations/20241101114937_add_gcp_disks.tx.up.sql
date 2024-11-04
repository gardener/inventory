CREATE TABLE IF NOT EXISTS "gcp_disk" (
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "zone" varchar NOT NULL,
    "region" varchar NOT NULL,
    "creation_timestamp" varchar,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_disk_key" UNIQUE ("name", "project_id")
);
