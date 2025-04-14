CREATE TABLE IF NOT EXISTS "l_openstack_loadbalancer_to_network" (
    lb_id UUID NOT NULL,
    network_id UUID NOT NULL,

    id UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "l_openstack_loadbalancer_to_network_pkey" PRIMARY KEY (id),
    CONSTRAINT "l_openstack_loadbalancer_to_network_key" UNIQUE (lb_id, network_id),
    CONSTRAINT "l_openstack_loadbalancer_to_network_lb_id_fkey" FOREIGN KEY ("lb_id") REFERENCES "openstack_loadbalancer" ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_loadbalancer_to_network_network_id_fkey" FOREIGN KEY ("network_id") REFERENCES "openstack_network" ("id") ON DELETE CASCADE
);
