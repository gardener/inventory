CREATE TABLE IF NOT EXISTS "g_project" (
    "name" VARCHAR NOT NULL,
    "namespace" VARCHAR NOT NULL,
    "status" VARCHAR NOT NULL,
    "purpose" VARCHAR NOT NULL,
    "owner" VARCHAR NOT NULL,
    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    UNIQUE ("name")
);