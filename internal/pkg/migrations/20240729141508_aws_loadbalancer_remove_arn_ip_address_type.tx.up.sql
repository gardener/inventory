ALTER TABLE IF EXISTS "aws_loadbalancer" DROP COLUMN "arn";
ALTER TABLE IF EXISTS "aws_loadbalancer" DROP COLUMN "ip_address_type";

ALTER TABLE "aws_loadbalancer" ADD CONSTRAINT "aws_loadbalancer_dns_name_unique" UNIQUE ("dns_name");
