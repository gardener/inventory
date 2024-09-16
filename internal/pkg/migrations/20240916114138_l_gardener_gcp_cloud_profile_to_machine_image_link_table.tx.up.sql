CREATE TABLE IF NOT EXISTS "l_g_gcp_image_to_cloud_profile" (
    "gcp_image_id" bigint NOT NULL,
    "cloud_profile_id" bigint NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("gcp_image_id") REFERENCES "g_cloud_profile_gcp_image" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("cloud_profile_id") REFERENCES "g_cloud_profile" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_g_gcp_image_to_cloud_profile_key" UNIQUE ("gcp_image_id", "cloud_profile_id")
);
