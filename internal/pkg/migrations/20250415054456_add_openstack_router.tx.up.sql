CREATE TABLE openstack_router (
    router_id varchar NOT NULL,
    name varchar NOT NULL,
    project_id varchar NOT NULL,
    domain varchar NOT NULL,
    region varchar NOT NULL,
    status varchar NOT NULL,
    description varchar NOT NULL,
    external_network_id varchar NOT NULL,

    id UUID DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "openstack_router_pkey" PRIMARY KEY (id),
    CONSTRAINT "openstack_router_key" UNIQUE ("router_id", "project_id")
);
