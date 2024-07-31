ALTER TABLE "aws_loadbalancer" ADD COLUMN "arn" VARCHAR NOT NULL DEFAULT '';
ALTER TABLE "aws_loadbalancer" ADD COLUMN "load_balancer_id" VARCHAR NOT NULL DEFAULT '';
