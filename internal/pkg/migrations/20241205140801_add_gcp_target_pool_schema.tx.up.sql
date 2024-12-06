-- Target pool
CREATE TABLE IF NOT EXISTS "gcp_target_pool" (
    "target_pool_id" bigint NOT NULL,
    "project_id" varchar NOT NULL,
    "name" varchar NOT NULL,
    "description" varchar NOT NULL,
    "backup_pool" varchar,
    "creation_timestamp" varchar,
    "region" varchar NOT NULL,
    "security_policy" varchar,
    "session_affinity" varchar NOT NULL,
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_target_pool_key" UNIQUE ("target_pool_id", "project_id")
);

-- Target pool instance
CREATE TABLE IF NOT EXISTS "gcp_target_pool_instance" (
    "target_pool_id" bigint NOT NULL,
    "project_id" varchar NOT NULL,
    "instance_name" varchar NOT NULL,
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    CONSTRAINT "gcp_target_pool_instance_key" UNIQUE ("target_pool_id", "project_id", "instance_name")
);

-- Target pool to instance
CREATE TABLE IF NOT EXISTS "l_gcp_target_pool_to_instance" (
    "target_pool_id" uuid NOT NULL,
    "instance_id" uuid NOT NULL,
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("target_pool_id") REFERENCES "gcp_target_pool" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("instance_id") REFERENCES "gcp_target_pool_instance" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_target_pool_to_instance_key" UNIQUE ("target_pool_id", "instance_id")
);

-- Target pool to project
CREATE TABLE IF NOT EXISTS "l_gcp_target_pool_to_project" (
    "target_pool_id" uuid NOT NULL,
    "project_id" uuid NOT NULL,
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("target_pool_id") REFERENCES "gcp_target_pool" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("project_id") REFERENCES "gcp_project" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_target_pool_to_project_key" UNIQUE ("target_pool_id", "project_id")
);
