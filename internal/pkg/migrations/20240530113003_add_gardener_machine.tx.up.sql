CREATE TABLE IF NOT EXISTS "g_machine" (
    "name" VARCHAR NOT NULL,
    "namespace" VARCHAR NOT NULL,
    "provider_id" VARCHAR NOT NULL,
    "status" VARCHAR NOT NULL,
    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    UNIQUE ("name"),
    UNIQUE ("provider_id")
);