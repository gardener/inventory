ALTER TABLE "aws_az" DROP CONSTRAINT "aws_az_zone_id_key";
ALTER TABLE "aws_az" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_az" ADD CONSTRAINT "aws_az_key" UNIQUE ("zone_id", "account_id");
