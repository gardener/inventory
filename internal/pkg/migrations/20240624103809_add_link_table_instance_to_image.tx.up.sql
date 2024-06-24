CREATE TABLE IF NOT EXISTS "l_aws_instance_to_image" (
    "instance_id" bigint NOT NULL,
    "image_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("instance_id") REFERENCES "aws_instance" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("image_id") REFERENCES "aws_image" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_instance_to_image_key" UNIQUE ("instance_id", "image_id")
);
