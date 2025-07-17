CREATE TABLE IF NOT EXISTS "openstack_volume" (
    "volume_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "project_id" varchar NOT NULL,
    "domain" varchar NOT NULL,
    "region" varchar NOT NULL,
    "user_id" varchar NOT NULL,
    "availability_zone" varchar NOT NULL,
    "size" int NOT NULL,
    "volume_type" varchar NOT NULL,
    "status" varchar NOT NULL,
    "replication_status" varchar NOT NULL,
    "bootable" boolean NOT NULL,
    "encrypted" boolean NOT NULL,
    "multi_attach" boolean NOT NULL,
    "snapshot_id" varchar NOT NULL,
    "description" varchar NOT NULL,
    "volume_created_at" timestamptz NOT NULL,
    "volume_updated_at" timestamptz NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    CONSTRAINT "openstack_volume_key" UNIQUE ("volume_id", "project_id", "domain", "region")
);
