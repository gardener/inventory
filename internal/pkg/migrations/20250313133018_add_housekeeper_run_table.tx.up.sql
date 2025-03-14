CREATE TABLE IF NOT EXISTS "aux_housekeeper_run" (
    "id" uuid NOT NULL DEFAULT gen_random_uuid (),
    "created_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "started_at" timestamptz NOT NULL,
    "completed_at" timestamptz NOT NULL,
    "is_ok" boolean NOT NULL,
    PRIMARY KEY ("id")
);

-- housekeeper_ran_in_last function returns TRUE when the housekeeper has
-- successfully ran in the last VAL interval, without encountering any errors.
CREATE FUNCTION housekeeper_ran_in_last(val interval)
RETURNS BOOLEAN AS
$func$
       SELECT EXISTS(
           SELECT
               is_ok
           FROM aux_housekeeper_run
           WHERE completed_at > NOW() - val AND is_ok IS TRUE
       );
$func$ LANGUAGE SQL STABLE;
