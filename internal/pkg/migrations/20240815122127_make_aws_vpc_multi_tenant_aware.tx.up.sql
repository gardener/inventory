ALTER TABLE "aws_vpc" DROP CONSTRAINT "aws_vpc_vpc_id_key";
ALTER TABLE "aws_vpc" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_vpc" ADD CONSTRAINT "aws_vpc_key" UNIQUE ("vpc_id", "account_id");
