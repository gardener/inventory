CREATE TABLE IF NOT EXISTS "l_gcp_addr_to_project" (
    "project_id" bigint NOT NULL,
    "address_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("project_id") REFERENCES "gcp_project" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("address_id") REFERENCES "gcp_address" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_addr_to_project_key" UNIQUE ("project_id", "address_id")
);
