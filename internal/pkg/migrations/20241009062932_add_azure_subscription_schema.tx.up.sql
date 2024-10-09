CREATE TABLE IF NOT EXISTS "az_subscription" (
    "subscription_id" varchar NOT NULL,
    "name" varchar,
    "state" varchar,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("subscription_id")
);
