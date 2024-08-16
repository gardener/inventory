ALTER TABLE "aws_bucket" DROP CONSTRAINT "aws_bucket_key";
ALTER TABLE "aws_bucket" DROP COLUMN "account_id";
ALTER TABLE "aws_bucket" ADD CONSTRAINT "aws_bucket_name_key" UNIQUE ("name");
