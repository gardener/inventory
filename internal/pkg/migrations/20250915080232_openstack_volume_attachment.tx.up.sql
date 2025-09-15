CREATE TABLE IF NOT EXISTS "openstack_volume_attachment" (
    "attachment_id" varchar NOT NULL,
    "volume_id" varchar NOT NULL,
    "attached_at" timestamptz NOT NULL,
    "device" varchar NOT NULL,
    "hostname" varchar NOT NULL,
    "server_id" varchar NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "l_openstack_volume_attachment_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "l_openstack_volume_attachment_key" UNIQUE ("attachment_id")
);
