CREATE TABLE IF NOT EXISTS "gcp_attached_disk" (
    "instance_name" varchar NOT NULL,
    "disk_name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "zone" varchar NOT NULL,
    "region" varchar NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_attached_disk_key" UNIQUE ("instance_name", "disk_name", "project_id")
);
