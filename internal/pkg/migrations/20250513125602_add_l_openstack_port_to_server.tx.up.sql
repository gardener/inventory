CREATE TABLE IF NOT EXISTS "l_openstack_port_to_server" (
    "port_id" UUID NOT NULL,
    "server_id" UUID NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "l_openstack_port_to_server_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "l_openstack_port_to_server_port_id_fkey" FOREIGN KEY ("port_id") REFERENCES openstack_port ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_port_to_server_server_id_fkey" FOREIGN KEY ("server_id") REFERENCES openstack_server ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_port_to_server_key" UNIQUE ("port_id", "server_id")
);
