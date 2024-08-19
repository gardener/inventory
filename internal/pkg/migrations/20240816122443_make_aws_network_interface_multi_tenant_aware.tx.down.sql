ALTER TABLE "aws_net_interface" DROP CONSTRAINT "aws_net_interface_key";
ALTER TABLE "aws_net_interface" DROP COLUMN "account_id";
ALTER TABLE "aws_net_interface" ADD CONSTRAINT "aws_net_interface_interface_id_key" UNIQUE ("interface_id");
