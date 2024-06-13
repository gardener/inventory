CREATE TABLE IF NOT EXISTS "l_g_shoot_to_seed" (
    "shoot_id" bigint NOT NULL,
    "seed_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("seed_id") REFERENCES "g_seed" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("shoot_id") REFERENCES "g_shoot" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_g_shoot_to_seed_key" UNIQUE ("shoot_id", "seed_id")
);
