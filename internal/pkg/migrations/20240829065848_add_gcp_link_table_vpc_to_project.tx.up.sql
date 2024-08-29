CREATE TABLE IF NOT EXISTS "l_gcp_vpc_to_project" (
    "project_id" bigint NOT NULL,
    "vpc_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("project_id") REFERENCES "gcp_project" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("vpc_id") REFERENCES "gcp_vpc" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_vpc_to_project_key" UNIQUE ("project_id", "vpc_id")
);
