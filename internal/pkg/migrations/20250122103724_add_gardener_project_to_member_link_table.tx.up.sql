CREATE TABLE IF NOT EXISTS "l_g_project_to_member" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "project_id" uuid NOT NULL,
    "member_id" uuid NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("project_id") REFERENCES "g_project" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("member_id") REFERENCES "g_project_member" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_g_project_to_member_key" UNIQUE ("project_id", "member_id")
);
