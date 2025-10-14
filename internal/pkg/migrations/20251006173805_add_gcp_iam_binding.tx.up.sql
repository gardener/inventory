CREATE TABLE IF NOT EXISTS "gcp_iam_binding" (
    "role" varchar NOT NULL,
    "resource_name" varchar NOT NULL,
    "resource_type" varchar NOT NULL,
    "condition" varchar,

    "id" uuid NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    UNIQUE ("role", "resource_name", "resource_type")
);
