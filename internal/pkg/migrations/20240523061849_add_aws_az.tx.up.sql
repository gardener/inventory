CREATE TABLE IF NOT EXISTS "aws_az" (
    "zone_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "opt_in_status" varchar NOT NULL,
    "state" varchar NOT NULL,
    "region_name" varchar NOT NULL,
    "group_name" varchar NOT NULL,
    "network_border_group" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("zone_id")
);
