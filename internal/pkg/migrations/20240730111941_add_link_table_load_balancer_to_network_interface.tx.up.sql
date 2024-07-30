CREATE TABLE IF NOT EXISTS "l_aws_lb_to_net_interface" (
    "lb_id" bigint NOT NULL,
    "ni_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("lb_id") REFERENCES "aws_loadbalancer" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("ni_id") REFERENCES "aws_net_interface" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_aws_lb_to_net_interface_key" UNIQUE ("lb_id", "ni_id")
);
