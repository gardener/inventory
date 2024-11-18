CREATE TABLE IF NOT EXISTS "g_persistent_volume" (
    "name" VARCHAR NOT NULL,
    "seed_name" VARCHAR NOT NULL,
    "provider" VARCHAR,
    "disk_ref" VARCHAR,
    "status" VARCHAR NOT NULL,
    "capacity" VARCHAR NOT NULL,
    "storage_class" VARCHAR NOT NULL,

    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    CONSTRAINT "g_persistent_volume_key" UNIQUE ("name", "seed_name")
);
