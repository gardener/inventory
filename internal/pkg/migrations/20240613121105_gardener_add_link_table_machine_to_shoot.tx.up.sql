CREATE TABLE IF NOT EXISTS "l_g_machine_to_shoot" (
    "shoot_id" bigint NOT NULL,
    "machine_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("shoot_id") REFERENCES "g_shoot" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("machine_id") REFERENCES "g_machine" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_g_machine_to_shoot_key" UNIQUE ("shoot_id", "machine_id")
);
