ALTER TABLE "aws_bucket" DROP CONSTRAINT "aws_bucket_name_key";
ALTER TABLE "aws_bucket" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_bucket" ADD CONSTRAINT "aws_bucket_key" UNIQUE ("name", "account_id");
