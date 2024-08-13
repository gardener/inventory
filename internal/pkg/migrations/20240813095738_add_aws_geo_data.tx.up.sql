CREATE TABLE IF NOT EXISTS "aws_geodata" (
    "id" bigserial NOT NULL,
    "region_name" varchar NOT NULL,
    "country_code" varchar NOT NULL,
    "city" varchar NOT NULL,
    "latitude" float NOT NULL,
    "longitude" float NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    UNIQUE ("region_name")
);
