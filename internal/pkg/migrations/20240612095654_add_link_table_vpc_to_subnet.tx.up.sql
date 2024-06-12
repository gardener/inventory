CREATE TABLE IF NOT EXISTS "l_aws_vpc_to_subnet" (
    "vpc_id" bigint NOT NULL,
    "subnet_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("vpc_id") REFERENCES "aws_vpc" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("subnet_id") REFERENCES "aws_subnet" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_vpc_to_subnet_key" UNIQUE ("vpc_id", "subnet_id")
);
