CREATE TABLE IF NOT EXISTS "openstack_subnet" (
    "subnet_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "region" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "network_id" varchar NOT NULL,
    "gateway_ip" varchar NOT NULL,
    "cidr" varchar NOT NULL,
    "subnet_pool_id" varchar NOT NULL,
    "enable_dhcp" boolean NOT NULL,
    "ip_version" integer NOT NULL,
    "description" varchar NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_subnet_key" UNIQUE ("subnet_id", "project_id")
);
