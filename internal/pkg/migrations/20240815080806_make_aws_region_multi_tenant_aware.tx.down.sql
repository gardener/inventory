ALTER TABLE "aws_region" DROP CONSTRAINT "aws_region_key";
ALTER TABLE "aws_region" DROP COLUMN "account_id";
ALTER TABLE "aws_region" ADD CONSTRAINT "aws_region_name_key" UNIQUE ("name");
