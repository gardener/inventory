ALTER TABLE "aws_image" DROP CONSTRAINT "aws_image_key";
ALTER TABLE "aws_image" DROP COLUMN "account_id";
ALTER TABLE "aws_image" ADD CONSTRAINT "aws_image_image_id_key" UNIQUE ("image_id");
