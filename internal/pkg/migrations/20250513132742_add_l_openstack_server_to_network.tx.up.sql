CREATE TABLE IF NOT EXISTS "l_openstack_server_to_network" (
    "server_id" UUID NOT NULL,
    "network_id" UUID NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "l_openstack_server_to_network_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "l_openstack_server_to_network_server_id_fkey" FOREIGN KEY ("server_id") REFERENCES openstack_server ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_server_to_network_network_id_fkey" FOREIGN KEY ("network_id") REFERENCES openstack_network ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_server_to_network_key" UNIQUE ("server_id", "network_id")
);
