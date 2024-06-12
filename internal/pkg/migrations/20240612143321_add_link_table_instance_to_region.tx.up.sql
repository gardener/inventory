CREATE TABLE IF NOT EXISTS "l_aws_instance_to_region" (
    "instance_id" bigint NOT NULL,
    "region_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("instance_id") REFERENCES "aws_instance" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("region_id") REFERENCES "aws_region" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_instance_to_region_key" UNIQUE ("instance_id", "region_id")
);
