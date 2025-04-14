CREATE TABLE IF NOT EXISTS "l_openstack_server_to_project" (
    server_id UUID NOT NULL,
    project_id UUID NOT NULL,

    id UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY ("id"),
    FOREIGN KEY ("server_id") REFERENCES "openstack_server" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("project_id") REFERENCES "openstack_project" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_server_to_project_key" UNIQUE ("server_id", "project_id")
);
