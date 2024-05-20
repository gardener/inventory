-- Modify "aws_region" table
ALTER TABLE "public"."aws_region" ALTER COLUMN "created_at" SET DEFAULT CURRENT_TIMESTAMP, ALTER COLUMN "updated_at" SET DEFAULT CURRENT_TIMESTAMP;
