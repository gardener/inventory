ALTER TABLE "gcp_disk" DROP CONSTRAINT "gcp_disk_key";
ALTER TABLE "gcp_disk" ADD COLUMN "description" VARCHAR NOT NULL DEFAULT '';
ALTER TABLE "gcp_disk" ADD COLUMN "type" VARCHAR NOT NULL DEFAULT '';
ALTER TABLE "gcp_disk" ADD COLUMN "is_regional" BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE "gcp_disk" ADD CONSTRAINT "gcp_disk_key" UNIQUE ("name", "project_id", "zone");
