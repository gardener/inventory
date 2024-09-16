ALTER TABLE "gcp_instance" DROP COLUMN "source_machine_image_link";
ALTER TABLE "gcp_instance" ADD COLUMN "source_machine_image" varchar NOT NULL DEFAULT '';
