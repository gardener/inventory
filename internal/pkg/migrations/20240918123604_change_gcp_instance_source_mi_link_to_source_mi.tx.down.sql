ALTER TABLE "gcp_instance" DROP COLUMN "source_machine_image";
ALTER TABLE "gcp_instance" ADD COLUMN "source_machine_image_link" varchar NOT NULL DEFAULT '';
