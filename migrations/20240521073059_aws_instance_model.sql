-- Create "aws_instance" table
CREATE TABLE "public"."aws_instance" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "name" text NULL,
  "arch" text NULL,
  "instance_id" text NULL,
  "instance_type" text NULL,
  "state" text NULL,
  "subnet_id" text NULL,
  "vpc_id" text NULL,
  "platform" text NULL,
  PRIMARY KEY ("id")
);
-- Create index "aws_instance_instance_id_idx" to table: "aws_instance"
CREATE UNIQUE INDEX "aws_instance_instance_id_idx" ON "public"."aws_instance" ("instance_id");
