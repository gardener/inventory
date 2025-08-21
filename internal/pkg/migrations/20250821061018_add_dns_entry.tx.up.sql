CREATE TABLE IF NOT EXISTS "g_dns_entry" (
    "name" VARCHAR NOT NULL,
    "namespace" VARCHAR NOT NULL,
    "fqdn" VARCHAR NOT NULL,
    "value" VARCHAR NOT NULL,
    "ttl" INTEGER,
    "dns_zone" VARCHAR NOT NULL,
    "provider_type" VARCHAR NOT NULL,
    "provider" VARCHAR NOT NULL,
    "seed_name" VARCHAR NOT NULL,
    "creation_timestamp" TIMESTAMPTZ,
    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "g_dns_entry_key" UNIQUE ("name", "namespace", "seed_name", "value")
);
