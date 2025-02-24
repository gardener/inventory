CREATE TABLE IF NOT EXISTS "l_openstack_loadbalancer_to_subnet" (
    lb_id UUID NOT NULL,
    subnet_id UUID NOT NULL,
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "l_openstack_loadbalancer_to_subnet_pkey" PRIMARY KEY (id),
    CONSTRAINT "l_openstack_loadbalancer_to_subnet_key" UNIQUE (lb_id, subnet_id),
    CONSTRAINT "l_openstack_loadbalancer_to_subnet_lb_id_fkey" FOREIGN KEY (lb_id) REFERENCES openstack_loadbalancer (id) ON DELETE CASCADE,
    CONSTRAINT "l_openstack_loadbalancer_to_subnet_subnet_id_fkey" FOREIGN KEY (subnet_id) REFERENCES openstack_subnet (id) ON DELETE CASCADE
);
