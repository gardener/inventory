CREATE TABLE IF NOT EXISTS "g_cloud_profile_aws_image" (
    "name" VARCHAR NOT NULL,
    "version" VARCHAR NOT NULL,
    "region_name" VARCHAR NOT NULL,
    "ami" VARCHAR NOT NULL UNIQUE,
    "architecture" VARCHAR NOT NULL,
    "cloud_profile_name" VARCHAR NOT NULL,

    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,

    PRIMARY KEY ("id"),
    CONSTRAINT "g_cloud_profile_aws_image_key" UNIQUE ("name", "version", "region_name", "ami")
);

