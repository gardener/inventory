CREATE TABLE IF NOT EXISTS "l_openstack_subnet_to_network" (
    subnet_id UUID NOT NULL,
    network_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "l_openstack_subnet_to_network_pkey" PRIMARY KEY (id),
    CONSTRAINT "l_openstack_subnet_to_network_key" UNIQUE (subnet_id, network_id),
    CONSTRAINT "l_openstack_subnet_to_network_subnet_id_fkey" FOREIGN KEY (subnet_id) REFERENCES openstack_subnet (id) ON DELETE CASCADE,
    CONSTRAINT "l_openstack_subnet_to_network_network_id_fkey" FOREIGN KEY (network_id) REFERENCES openstack_network (id) ON DELETE CASCADE
);
