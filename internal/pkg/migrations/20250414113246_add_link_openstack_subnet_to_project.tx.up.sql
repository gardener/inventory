CREATE TABLE "l_openstack_subnet_to_project" (
    subnet_id UUID NOT NULL,
    project_id UUID NOT NULL,

    id UUID DEFAULT gen_random_uuid(),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT "l_openstack_subnet_to_project_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "l_openstack_subnet_to_project_subnet_id_fkey" FOREIGN KEY ("subnet_id") REFERENCES openstack_subnet ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_subnet_to_project_project_id_fkey" FOREIGN KEY ("project_id") REFERENCES openstack_project ("id") ON DELETE CASCADE,
    CONSTRAINT "l_openstack_subnet_to_project_key" UNIQUE ("subnet_id", "project_id")
);
