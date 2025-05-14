CREATE TABLE IF NOT EXISTS "l_openstack_server_to_port" (
    "server_id" UUID NOT NULL,
    "port_id" UUID NOT NULL,

    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "l_openstack_server_to_port_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "l_openstack_server_to_port_server_id_fkey" FOREIGN KEY ("server_id") REFERENCES openstack_server ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_server_to_port_port_id_fkey" FOREIGN KEY ("port_id") REFERENCES openstack_port ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_server_to_port_key" UNIQUE ("server_id", "port_id")
);
