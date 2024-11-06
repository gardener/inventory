CREATE TABLE IF NOT EXISTS "l_gcp_gke_cluster_to_project" (
    "cluster_id" bigint NOT NULL,
    "project_id" bigint NOT NULL,
    "id" bigserial NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("cluster_id") REFERENCES "gcp_gke_cluster" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("project_id") REFERENCES "gcp_project" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_gcp_gke_cluster_to_project_key" UNIQUE ("cluster_id", "project_id")
);
