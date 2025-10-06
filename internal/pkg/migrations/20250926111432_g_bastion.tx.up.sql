CREATE TABLE IF NOT EXISTS "g_bastion" (
    "name" varchar NOT NULL,
    "namespace" varchar NOT NULL,
    "seed_name" varchar NOT NULL,
    "ip" varchar,
    "hostname" varchar,

    "id" UUID DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "g_bastion_pkey" PRIMARY KEY (id),
    CONSTRAINT "g_bastion_key" UNIQUE ("name", "namespace", "seed_name")
)
