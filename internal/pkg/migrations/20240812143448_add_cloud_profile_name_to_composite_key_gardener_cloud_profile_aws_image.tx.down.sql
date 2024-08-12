ALTER TABLE "g_cloud_profile_aws_image" DROP CONSTRAINT "g_cloud_profile_aws_image_key";
ALTER TABLE "g_cloud_profile_aws_image" ADD CONSTRAINT "g_cloud_profile_aws_image_key" UNIQUE ("name", "version", "region_name", "ami")
