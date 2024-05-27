CREATE TABLE IF NOT EXISTS "aws_subnet" (
    "name" varchar NOT NULL,
    "subnet_id" varchar NOT NULL,
    "vpc_id" varchar NOT NULL,
    "state" varchar NOT NULL,
    "az" varchar NOT NULL,
    "az_id" varchar NOT NULL,
    "available_ipv4_addresses" bigint NOT NULL,
    "ipv4_cidr" varchar NOT NULL,
    "ipv6_cidr" varchar,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("subnet_id")
);
