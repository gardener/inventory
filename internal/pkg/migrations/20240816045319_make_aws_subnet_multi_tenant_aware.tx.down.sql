ALTER TABLE "aws_subnet" DROP CONSTRAINT "aws_subnet_key";
ALTER TABLE "aws_subnet" DROP COLUMN "account_id";
ALTER TABLE "aws_subnet" ADD CONSTRAINT "aws_subnet_subnet_id_key" UNIQUE ("subnet_id");
