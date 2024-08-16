ALTER TABLE "aws_image" DROP CONSTRAINT "aws_image_image_id_key";
ALTER TABLE "aws_image" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_image" ADD CONSTRAINT "aws_image_key" UNIQUE ("image_id", "account_id");
