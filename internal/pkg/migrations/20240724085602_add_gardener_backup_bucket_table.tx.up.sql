CREATE TABLE IF NOT EXISTS "g_backup_bucket" (
    "name" VARCHAR UNIQUE NOT NULL,
    "provider_id" VARCHAR NOT NULL,
    "region_name" VARCHAR NOT NULL,
    "seed_name" VARCHAR NOT NULL,

    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,

    PRIMARY KEY ("id")
);
