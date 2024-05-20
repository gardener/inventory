-- Create "aws_az" table
CREATE TABLE "public"."aws_az" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamptz NULL DEFAULT CURRENT_TIMESTAMP,
  "name" text NULL,
  "zone_id" text NULL,
  "opt_in_status" text NULL,
  "state" text NULL,
  "region_name" text NULL,
  "group_name" text NULL,
  "network_border_group" text NULL,
  PRIMARY KEY ("id")
);
-- Create index "aws_az_zone_id_idx" to table: "aws_az"
CREATE UNIQUE INDEX "aws_az_zone_id_idx" ON "public"."aws_az" ("zone_id");
