ALTER TABLE "aws_region" DROP CONSTRAINT "aws_region_name_key";
ALTER TABLE "aws_region" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_region" ADD CONSTRAINT "aws_region_key" UNIQUE ("name", "account_id");
