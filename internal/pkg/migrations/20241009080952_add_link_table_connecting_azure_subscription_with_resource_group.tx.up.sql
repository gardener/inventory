CREATE TABLE IF NOT EXISTS "l_az_rg_to_subscription" (
    "rg_id" bigint NOT NULL,
    "sub_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("rg_id") REFERENCES "az_resource_group" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("sub_id") REFERENCES "az_subscription" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_az_rg_to_subscription_key" UNIQUE ("rg_id", "sub_id")
);
