ALTER TABLE gcp_address ALTER COLUMN address TYPE varchar USING address::text;
