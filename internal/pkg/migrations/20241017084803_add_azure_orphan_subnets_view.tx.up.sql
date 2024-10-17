CREATE OR REPLACE VIEW az_orphan_subnet AS
SELECT
        s.name,
        s.subscription_id,
        s.resource_group,
        s.provisioning_state,
        s.vpc_name,
        s.address_prefix,
        s.security_group,
        s.purpose,
        s.created_at,
        s.updated_at,
        v.location
FROM az_subnet AS s
INNER JOIN az_orphan_vpc AS v
      ON s.vpc_name = v.name AND s.subscription_id = v.subscription_id AND s.resource_group = v.resource_group;
