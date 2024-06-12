CREATE TABLE IF NOT EXISTS "l_aws_region_to_vpc" (
    "region_id" bigint NOT NULL,
    "vpc_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "l_aws_region_to_vpc_key" UNIQUE ("region_id", "vpc_id"),
    FOREIGN KEY ("region_id") REFERENCES "aws_region" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("vpc_id") REFERENCES "aws_vpc" ("id") ON DELETE CASCADE
);
