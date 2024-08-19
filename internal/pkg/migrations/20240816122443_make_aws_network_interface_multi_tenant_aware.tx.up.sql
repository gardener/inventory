ALTER TABLE "aws_net_interface" DROP CONSTRAINT "aws_net_interface_interface_id_key";
ALTER TABLE "aws_net_interface" ADD COLUMN "account_id" VARCHAR NOT NULL;
ALTER TABLE "aws_net_interface" ADD CONSTRAINT "aws_net_interface_key" UNIQUE ("interface_id", "account_id");
