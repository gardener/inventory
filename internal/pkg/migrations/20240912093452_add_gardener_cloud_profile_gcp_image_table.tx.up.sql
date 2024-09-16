CREATE TABLE IF NOT EXISTS "g_cloud_profile_gcp_image" (
    "name" VARCHAR NOT NULL,
    "version" VARCHAR NOT NULL,
    "image" VARCHAR NOT NULL,
    "architecture" VARCHAR NOT NULL,
    "cloud_profile_name" VARCHAR NOT NULL,

    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,

    PRIMARY KEY ("id"),
    CONSTRAINT "g_cloud_profile_gcp_image_key" UNIQUE ("name", "image", "version", "cloud_profile_name")
);
