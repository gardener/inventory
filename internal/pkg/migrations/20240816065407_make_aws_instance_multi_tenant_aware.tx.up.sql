ALTER TABLE "aws_instance" DROP CONSTRAINT "aws_instance_instance_id_key";
ALTER TABLE "aws_instance" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_instance" ADD CONSTRAINT "aws_instance_key" UNIQUE ("instance_id", "account_id");
