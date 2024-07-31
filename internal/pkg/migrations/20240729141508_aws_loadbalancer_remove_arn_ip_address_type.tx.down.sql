ALTER TABLE IF EXISTS "aws_loadbalancer" ADD COLUMN "arn" VARCHAR DEFAULT '';
ALTER TABLE IF EXISTS "aws_loadbalancer" ADD COLUMN "ip_address_type" VARCHAR DEFAULT '';

ALTER TABLE "aws_loadbalancer" DROP CONSTRAINT "aws_loadbalancer_dns_name_key";
