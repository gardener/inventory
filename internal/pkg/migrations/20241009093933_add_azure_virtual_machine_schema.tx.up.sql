CREATE TABLE IF NOT EXISTS "az_vm" (
    "name" varchar NOT NULL,
    "subscription_id" varchar NOT NULL,
    "resource_group" varchar NOT NULL,
    "location" varchar NOT NULL,
    "provisioning_state" varchar NOT NULL,
    "vm_created_at" timestamptz,
    "hyper_v_gen" varchar,
    "vm_size" varchar,
    "power_state" varchar,
    "vm_agent_version" varchar,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "az_vm_key" UNIQUE ("name", "subscription_id", "resource_group")
);
