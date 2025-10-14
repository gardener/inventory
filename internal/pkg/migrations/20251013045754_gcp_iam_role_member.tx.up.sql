CREATE TABLE IF NOT EXISTS "gcp_iam_role_member" (
    "role" varchar NOT NULL,
    "member" varchar NOT NULL,
    "resource_name" varchar NOT NULL,
    "resource_type" varchar NOT NULL,

    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_iam_role_member_key" UNIQUE ("role", "member", "resource_name", "resource_type")
);
