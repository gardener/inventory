ALTER TABLE "aws_vpc" DROP CONSTRAINT "aws_vpc_key";
ALTER TABLE "aws_vpc" DROP COLUMN "account_id";
ALTER TABLE "aws_vpc" ADD CONSTRAINT "aws_vpc_vpc_id_key" UNIQUE ("vpc_id");
