ALTER TABLE "gcp_disk" DROP CONSTRAINT "gcp_disk_key";
ALTER TABLE "gcp_disk" DROP COLUMN "description";
ALTER TABLE "gcp_disk" DROP COLUMN "type";
ALTER TABLE "gcp_disk" DROP COLUMN "is_regional";
ALTER TABLE "gcp_disk" ADD CONSTRAINT "gcp_disk_key" UNIQUE ("name", "project_id");
