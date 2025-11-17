CREATE TABLE IF NOT EXISTS "az_network_interface" (
    "name" varchar NOT NULL,
    "subscription_id" varchar NOT NULL,
    "resource_group" varchar NOT NULL,
    "location" varchar NOT NULL,
    "provisioning_state" varchar NOT NULL,
    "mac_address" varchar,
    "nic_type" varchar,
    "primary_nic" boolean NOT NULL,
    "vm_name" varchar,
    "vpc_name" varchar,
    "subnet_name" varchar,
    "private_ip" inet,
    "private_ip_allocation" varchar,
    "public_ip_name" varchar,
    "network_security_group" varchar,
    "ip_forwarding_enabled" boolean NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "az_network_interface_key" UNIQUE ("name", "subscription_id", "resource_group")
);
