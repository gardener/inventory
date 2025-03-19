CREATE TABLE IF NOT EXISTS "g_cloud_profile_openstack_image" (
    "name" VARCHAR NOT NULL,
    "version" VARCHAR NOT NULL,
    "region_name" VARCHAR NOT NULL,
    "image_id" VARCHAR NOT NULL,
    "architecture" VARCHAR NOT NULL,
    "cloud_profile_name" VARCHAR NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,

    PRIMARY KEY ("id"),
    CONSTRAINT "g_cloud_profile_openstack_image_key" UNIQUE ("name", "version", "region_name", "image_id", "cloud_profile_name")
);
