ALTER TABLE "aws_loadbalancer" DROP CONSTRAINT "aws_loadbalancer_key";
ALTER TABLE "aws_loadbalancer" DROP COLUMN "account_id";
ALTER TABLE "aws_loadbalancer" ADD CONSTRAINT "aws_loadbalancer_dns_name_key" UNIQUE ("dns_name");
