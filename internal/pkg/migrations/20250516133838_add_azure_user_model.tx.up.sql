CREATE TABLE IF NOT EXISTS "az_user" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "user_id" varchar NOT NULL,
    "tenant_id" varchar NOT NULL,
    "mail" varchar NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "az_user_key" UNIQUE ("user_id", "tenant_id")
);
