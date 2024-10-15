CREATE TABLE IF NOT EXISTS "az_subnet" (
    "name" varchar NOT NULL,
    "subscription_id" varchar NOT NULL,
    "resource_group" varchar NOT NULL,
    "type" varchar NOT NULL ,
    "provisioning_state" varchar NOT NULL,
    "vpc_name" varchar NOT NULL ,
    "address_prefix" varchar NOT NULL,
    "security_group" varchar NOT NULL ,
    "purpose" varchar NOT NULL ,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "az_subnet_key" UNIQUE ("name", "vpc_name", "subscription_id", "resource_group")
);
