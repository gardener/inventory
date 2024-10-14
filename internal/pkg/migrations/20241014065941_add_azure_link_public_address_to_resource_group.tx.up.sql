CREATE TABLE IF NOT EXISTS "l_az_pub_addr_to_rg" (
    "rg_id" bigint NOT NULL,
    "pa_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("rg_id") REFERENCES "az_resource_group" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("pa_id") REFERENCES "az_public_address" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_az_pub_addr_to_rg_key" UNIQUE ("rg_id", "pa_id")
);
