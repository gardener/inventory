CREATE TABLE IF NOT EXISTS "aws_bucket" (
    "name" varchar NOT NULL,
    "creation_date" timestamptz NOT NULL,
    "region_name" varchar NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    UNIQUE ("name")
);
