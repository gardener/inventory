CREATE TABLE IF NOT EXISTS "az_lb" (
    "name" varchar NOT NULL,
    "subscription_id" varchar NOT NULL,
    "resource_group" varchar NOT NULL,
    "location" varchar NOT NULL,
    "provisioning_state" varchar NOT NULL,
    "sku_name" varchar NOT NULL,
    "sku_tier" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "az_lb_key" UNIQUE ("name", "subscription_id", "resource_group")
);
