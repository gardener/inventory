ALTER TABLE "aws_loadbalancer" DROP CONSTRAINT "aws_loadbalancer_dns_name_key";
ALTER TABLE "aws_loadbalancer" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_loadbalancer" ADD CONSTRAINT "aws_loadbalancer_key" UNIQUE ("dns_name", "account_id");
