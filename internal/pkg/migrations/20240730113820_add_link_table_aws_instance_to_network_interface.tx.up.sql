CREATE TABLE IF NOT EXISTS "l_aws_instance_to_net_interface" (
    "instance_id" bigint NOT NULL,
    "ni_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("instance_id") REFERENCES "aws_instance" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("ni_id") REFERENCES "aws_net_interface" ("id") ON DELETE CASCADE
    CONSTRAINT "l_aws_instance_to_net_interface_key" UNIQUE ("instance_id", "ni_id")
);
