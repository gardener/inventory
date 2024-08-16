ALTER TABLE "aws_instance" DROP CONSTRAINT "aws_instance_key";
ALTER TABLE "aws_instance" DROP COLUMN "account_id";
ALTER TABLE "aws_instance" ADD CONSTRAINT "aws_instance_instance_id_key" UNIQUE ("instance_id");
