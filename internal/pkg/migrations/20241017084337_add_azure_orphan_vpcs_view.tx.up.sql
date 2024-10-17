CREATE OR REPLACE VIEW az_orphan_vpc AS
SELECT
        v.name,
        v.subscription_id,
        v.resource_group,
        v.location,
        v.provisioning_state,
        v.encryption_enabled,
        v.vm_protection_enabled,
        v.created_at,
        v.updated_at
FROM az_vpc AS v
LEFT JOIN g_shoot AS s ON v.resource_group = s.technical_id
WHERE s.name IS NULL;
