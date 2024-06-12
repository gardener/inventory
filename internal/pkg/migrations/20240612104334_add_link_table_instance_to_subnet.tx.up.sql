CREATE TABLE IF NOT EXISTS "l_aws_instance_to_subnet" (
    "instance_id" bigint NOT NULL,
    "subnet_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("instance_id") REFERENCES "aws_instance" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("subnet_id") REFERENCES "aws_subnet" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_instance_to_subnet_key" UNIQUE ("instance_id", "subnet_id")
);
