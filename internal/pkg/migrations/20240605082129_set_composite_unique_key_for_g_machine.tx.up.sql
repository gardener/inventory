ALTER TABLE g_machine DROP CONSTRAINT g_machine_name_key;
ALTER TABLE g_machine DROP CONSTRAINT g_machine_provider_id_key;
ALTER TABLE g_machine ADD CONSTRAINT g_machine_name_namespace_key UNIQUE (name, namespace);
