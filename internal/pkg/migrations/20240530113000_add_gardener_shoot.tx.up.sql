CREATE TABLE IF NOT EXISTS "g_shoot" (
    "name" VARCHAR NOT NULL,
    "technical_id" VARCHAR NOT NULL,
    "namespace" VARCHAR NOT NULL,
    "project_name" VARCHAR NOT NULL,
    "cloud_profile" VARCHAR NOT NULL,
    "purpose" VARCHAR NOT NULL,
    "seed_name" VARCHAR NOT NULL,
    "status" VARCHAR NOT NULL,
    "is_hibernated" BOOLEAN NOT NULL,
    "created_by" VARCHAR NOT NULL,
    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    UNIQUE ("technical_id")
);