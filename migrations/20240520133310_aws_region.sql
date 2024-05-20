-- Create "aws_region" table
CREATE TABLE "public"."aws_region" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "name" text NULL,
  "endpoint" text NULL,
  "opt_in_status" text NULL,
  PRIMARY KEY ("id")
);
-- Create index "aws_region_name_idx" to table: "aws_region"
CREATE UNIQUE INDEX "aws_region_name_idx" ON "public"."aws_region" ("name");
