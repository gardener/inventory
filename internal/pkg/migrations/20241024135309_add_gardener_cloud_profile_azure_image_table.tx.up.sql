CREATE TABLE IF NOT EXISTS "g_cloud_profile_azure_image" (
    "name" VARCHAR NOT NULL,
    "version" VARCHAR NOT NULL,
    "urn" VARCHAR NOT NULL,
    "architecture" VARCHAR NOT NULL,
    "cloud_profile_name" VARCHAR NOT NULL,

    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,

    PRIMARY KEY ("id"),
    CONSTRAINT "g_cloud_profile_azure_image_key" UNIQUE ("name", "urn", "version", "cloud_profile_name")
);

