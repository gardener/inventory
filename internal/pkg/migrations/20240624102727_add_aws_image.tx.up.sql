CREATE TABLE IF NOT EXISTS "aws_image" (
    "name" varchar NOT NULL,
    "owner_id" varchar NOT NULL,
    "image_id" varchar NOT NULL,
    "image_type" varchar NOT NULL,
    "source" varchar NOT NULL,
    "root_device_type" varchar NOT NULL,
    "description" varchar NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("image_id")
);
