ALTER TABLE gcp_disk ADD COLUMN last_attach_timestamp VARCHAR;
ALTER TABLE gcp_disk ADD COLUMN last_detach_timestamp VARCHAR;
ALTER TABLE gcp_disk ADD COLUMN status VARCHAR;
ALTER TABLE gcp_disk ADD COLUMN size_gb INT NOT NULL DEFAULT 0;
