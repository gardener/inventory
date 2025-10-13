CREATE TABLE IF NOT EXISTS "gcp_iam_policy" (
    "resource_name" varchar NOT NULL,
    "resource_type" varchar NOT NULL,
    "version" integer NOT NULL,
    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("resource_name", "resource_type")
);
