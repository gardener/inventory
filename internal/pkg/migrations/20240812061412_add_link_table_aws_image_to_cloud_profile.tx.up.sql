CREATE TABLE IF NOT EXISTS "l_g_aws_image_to_cloud_profile" (
    "aws_image_id" bigint NOT NULL,
    "cloud_profile_id" bigint NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("aws_image_id") REFERENCES "g_cloud_profile_aws_image" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("cloud_profile_id") REFERENCES "g_cloud_profile" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_g_aws_image_to_cloud_profile_key" UNIQUE ("aws_image_id", "cloud_profile_id")
);
