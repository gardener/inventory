CREATE TABLE IF NOT EXISTS "az_blob_container" (
    "name" varchar NOT NULL,
    "subscription_id" varchar NOT NULL,
    "resource_group" varchar NOT NULL,
    "storage_account" varchar NOT NULL,
    "public_access" varchar NOT NULL,
    "deleted" boolean NOT NULL,
    "last_modified_time" timestamptz,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "az_blob_container_key" UNIQUE ("name", "storage_account", "resource_group", "subscription_id")
);
