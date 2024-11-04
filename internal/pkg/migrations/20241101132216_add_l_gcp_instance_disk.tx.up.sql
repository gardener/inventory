CREATE TABLE IF NOT EXISTS "l_gcp_instance_to_disk" (
    "instance_id" bigint NOT NULL,
    "disk_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("instance_id") REFERENCES "gcp_instance" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("disk_id") REFERENCES "gcp_disk" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_instance_to_disk_key" UNIQUE ("instance_id", "disk_id")
);
