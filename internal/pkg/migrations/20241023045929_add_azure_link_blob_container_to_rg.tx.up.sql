CREATE TABLE IF NOT EXISTS "l_az_blob_container_to_rg" (
    "blob_container_id" bigint NOT NULL,
    "rg_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("blob_container_id") REFERENCES "az_blob_container" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("rg_id") REFERENCES "az_resource_group" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_az_blob_container_to_rg_key" UNIQUE ("blob_container_id", "rg_id")
);
