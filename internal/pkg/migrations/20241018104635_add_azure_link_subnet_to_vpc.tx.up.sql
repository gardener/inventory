CREATE TABLE IF NOT EXISTS "l_az_subnet_to_vpc" (
    "subnet_id" bigint NOT NULL,
    "vpc_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("subnet_id") REFERENCES "az_subnet" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("vpc_id") REFERENCES "az_vpc" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_az_subnet_to_vpc_key" UNIQUE ("subnet_id", "vpc_id")
);
