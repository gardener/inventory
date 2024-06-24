CREATE TABLE IF NOT EXISTS "l_aws_image_to_region" (
    "image_id" bigint NOT NULL,
    "region_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("image_id") REFERENCES "aws_image" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("region_id") REFERENCES "aws_region" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_image_to_region_key" UNIQUE ("image_id", "region_id")
);
