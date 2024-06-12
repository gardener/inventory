CREATE TABLE IF NOT EXISTS "l_aws_subnet_to_az" (
    "az_id" bigint NOT NULL,
    "subnet_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("az_id") REFERENCES "aws_az" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("subnet_id") REFERENCES "aws_subnet" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_subnet_to_az_key" UNIQUE ("az_id", "subnet_id")
);
