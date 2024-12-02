ALTER TABLE "g_project" ADD COLUMN creation_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE "g_seed" ADD COLUMN creation_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE "g_machine" ADD COLUMN creation_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE "g_backup_bucket" ADD COLUMN creation_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE "g_cloud_profile" ADD COLUMN creation_timestamp TIMESTAMP WITH TIME ZONE;
ALTER TABLE "g_persistent_volume" ADD COLUMN creation_timestamp TIMESTAMP WITH TIME ZONE;
