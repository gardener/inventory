CREATE TABLE IF NOT EXISTS "l_aws_lb_to_vpc" (
    "lb_id" bigint NOT NULL,
    "vpc_id" bigint NOT NULL,

    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    FOREIGN KEY ("lb_id") REFERENCES "aws_loadbalancer" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("vpc_id") REFERENCES "aws_vpc" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_lb_to_vpc_key" UNIQUE ("lb_id", "vpc_id")
);
