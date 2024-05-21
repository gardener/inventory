-- Create "aws_vpc" table
CREATE TABLE "public"."aws_vpc" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "name" text NULL,
  "vpc_id" text NULL,
  "state" text NULL,
  "ipv4_c_id_r" text NULL,
  "ipv6_c_id_r" text NULL,
  "is_default" boolean NULL,
  "owner_id" text NULL,
  "region_name" text NULL,
  PRIMARY KEY ("id")
);
-- Create index "aws_vpc_vpc_id_idx" to table: "aws_vpc"
CREATE UNIQUE INDEX "aws_vpc_vpc_id_idx" ON "public"."aws_vpc" ("vpc_id");
