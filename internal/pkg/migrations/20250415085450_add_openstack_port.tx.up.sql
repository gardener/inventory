CREATE TABLE "openstack_port" (
    port_id varchar NOT NULL ,
    name varchar NOT NULL,
    project_id varchar NOT NULL,
    network_id varchar NOT NULL,
    device_id varchar NOT NULL,
    device_owner varchar NOT NULL,
    domain varchar NOT NULL,
    region varchar NOT NULL,
    mac_address varchar NOT NULL,
    status varchar NOT NULL,
    description varchar NOT NULL,
    port_created_at timestamptz NOT NULL,
    port_updated_at timestamptz NOT NULL,

    id UUID DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "openstack_port_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "openstack_port_key" UNIQUE ("port_id", "project_id",  "network_id", "region")
);
