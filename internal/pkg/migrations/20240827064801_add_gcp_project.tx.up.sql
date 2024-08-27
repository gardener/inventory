CREATE TABLE IF NOT EXISTS "gcp_project" (
    "name" varchar NOT NULL,
    "parent" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "state" varchar NOT NULL,
    "display_name" varchar NOT NULL,
    "project_create_time" timestamptz,
    "project_update_time" timestamptz,
    "project_delete_time" timestamptz,
    "etag" varchar NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("project_id")
);
