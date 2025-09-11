CREATE TABLE IF NOT EXISTS "aws_dhcp_option_set" (
    "name" varchar NOT NULL,
    "set_id" varchar NOT NULL,
    "account_id" varchar NOT NULL,
    "region_name" varchar NOT NULL,

    "id" UUID DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "aws_dhcp_option_set_key" UNIQUE ("set_id", "account_id")
);
