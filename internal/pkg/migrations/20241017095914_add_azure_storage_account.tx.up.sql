CREATE TABLE IF NOT EXISTS "az_storage_account" (
    "name" varchar NOT NULL,
    "subscription_id" varchar NOT NULL,
    "resource_group" varchar NOT NULL,
    "location" varchar NOT NULL,
    "provisioning_state" varchar NOT NULL,
    "kind" varchar NOT NULL,
    "sku_name" varchar NOT NULL ,
    "sku_tier" varchar NOT NULL ,
    "creation_time" timestamptz,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "az_storage_account_key" UNIQUE ("name", "resource_group", "subscription_id")
);
