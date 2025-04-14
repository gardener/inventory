CREATE TABLE IF NOT EXISTS "l_openstack_network_to_project" (
    network_id UUID NOT NULL,
    project_id UUID NOT NULL,

    id UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "l_openstack_network_to_project_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "l_openstack_network_to_project_network_id_fkey" FOREIGN KEY ("network_id") REFERENCES "openstack_network" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_network_to_project_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES "openstack_project" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_network_to_project_key" UNIQUE ("network_id", "project_id")
);