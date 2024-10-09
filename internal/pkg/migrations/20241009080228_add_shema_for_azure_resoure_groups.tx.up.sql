CREATE TABLE IF NOT EXISTS "az_resource_group" (
    "name" varchar NOT NULL,
    "subscription_id" varchar NOT NULL,
    "location" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "az_resource_group_key" UNIQUE ("name", "subscription_id")
);
