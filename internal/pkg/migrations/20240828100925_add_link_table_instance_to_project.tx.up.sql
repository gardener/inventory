CREATE TABLE IF NOT EXISTS "l_gcp_instance_to_project" (
    "project_id" bigint NOT NULL,
    "instance_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("project_id") REFERENCES "gcp_project" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("instance_id") REFERENCES "gcp_instance" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_instance_to_project_key" UNIQUE ("project_id", "instance_id")
);
