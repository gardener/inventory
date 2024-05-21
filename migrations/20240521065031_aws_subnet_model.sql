-- Create "aws_subnet" table
CREATE TABLE "public"."aws_subnet" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "name" text NULL,
  "subnet_id" text NULL,
  "vpc_id" text NULL,
  "state" text NULL,
  "az" text NULL,
  "az_id" text NULL,
  "available_ipv4_addresses" bigint NULL,
  "ipv4_c_id_r" text NULL,
  "ipv6_c_id_r" text NULL,
  PRIMARY KEY ("id")
);
-- Create index "aws_subnet_subnet_id_idx" to table: "aws_subnet"
CREATE UNIQUE INDEX "aws_subnet_subnet_id_idx" ON "public"."aws_subnet" ("subnet_id");
