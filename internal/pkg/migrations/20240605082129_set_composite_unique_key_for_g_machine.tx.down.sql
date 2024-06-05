ALTER TABLE g_machine ADD CONSTRAINT g_machine_name_key UNIQUE (name);
ALTER TABLE g_machine ADD CONSTRAINT g_machine_provider_id_key UNIQUE (provider_id);
ALTER TABLE g_machine DROP CONSTRAINT g_machine_name_namespace_key;
