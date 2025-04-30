CREATE TABLE IF NOT EXISTS "openstack_port_ip" (
    port_id varchar NOT NULL,
    ip_address inet NOT NULL,
    subnet_id varchar NOT NULL,
    
    id UUID DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "openstack_port_ip_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "openstack_port_ip_key" UNIQUE ("port_id", "ip_address", "subnet_id")
);
