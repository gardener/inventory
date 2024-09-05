CREATE TABLE IF NOT EXISTS "l_gcp_instance_to_nic" (
    "instance_id" bigint NOT NULL,
    "nic_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("instance_id") REFERENCES "gcp_instance" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("nic_id") REFERENCES "gcp_nic" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_instance_to_nic_key" UNIQUE ("instance_id", "nic_id")
);
