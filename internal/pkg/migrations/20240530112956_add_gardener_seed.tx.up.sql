CREATE TABLE IF NOT EXISTS "g_seed" (
    "name" VARCHAR NOT NULL,
    "kubernetes_version" VARCHAR NOT NULL,
    "id" BIGSERIAL NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    UNIQUE ("name")
);