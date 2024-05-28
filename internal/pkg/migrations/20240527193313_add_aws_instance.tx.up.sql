CREATE TABLE IF NOT EXISTS "aws_instance" (
    "name" varchar NOT NULL,
    "arch" varchar NOT NULL,
    "instance_id" varchar NOT NULL,
    "instance_type" varchar NOT NULL,
    "state" varchar NOT NULL,
    "subnet_id" varchar NOT NULL,
    "vpc_id" varchar NOT NULL,
    "platform" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("instance_id")
);
