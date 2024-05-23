CREATE TABLE IF NOT EXISTS "aws_vpc" (
    "name" varchar NOT NULL,
    "vpc_id" varchar NOT NULL,
    "state" varchar NOT NULL,
    "ipv4_cidr" varchar NOT NULL,
    "ipv6_cidr" varchar,
    "is_default" boolean NOT NULL,
    "owner_id" varchar NOT NULL,
    "region_name" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("vpc_id")
);
