CREATE TABLE IF NOT EXISTS "gcp_bucket" (
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "location_type" varchar NOT NULL,
    "location" varchar NOT NULL,
    "default_storage_class" varchar NOT NULL,
    "creation_timestamp" varchar,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_bucket_key" UNIQUE ("name", "project_id")
);
