ALTER TABLE "aws_subnet" DROP CONSTRAINT "aws_subnet_subnet_id_key";
ALTER TABLE "aws_subnet" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_subnet" ADD CONSTRAINT "aws_subnet_key" UNIQUE ("subnet_id", "account_id");
