ALTER TABLE "aws_az" DROP CONSTRAINT "aws_az_key";
ALTER TABLE "aws_az" DROP COLUMN "account_id";
ALTER TABLE "aws_az" ADD CONSTRAINT "aws_az_zone_id_key" UNIQUE ("zone_id");
