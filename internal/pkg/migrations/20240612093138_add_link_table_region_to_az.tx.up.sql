CREATE TABLE IF NOT EXISTS "l_aws_region_to_az" (
    "region_id" bigint NOT NULL,
    "az_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("region_id") REFERENCES "aws_region" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("az_id") REFERENCES "aws_az" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_region_to_az_key" UNIQUE ("region_id", "az_id")
);
