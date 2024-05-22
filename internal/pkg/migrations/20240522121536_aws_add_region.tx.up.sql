CREATE TABLE IF NOT EXISTS "aws_region" (
    "name" varchar NOT NULL,
    "endpoint" varchar NOT NULL,
    "opt_in_status" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("name")
);
