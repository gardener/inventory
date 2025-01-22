CREATE TABLE IF NOT EXISTS "g_project_member" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" varchar NOT NULL,
    "project_name" varchar NOT NULL,
    "kind" varchar NOT NULL,
    "role" varchar NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "g_project_member_key" UNIQUE ("name", "project_name")
);
