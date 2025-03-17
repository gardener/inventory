DROP FUNCTION housekeeper_ran_in_last;
DELETE FROM "aux_housekeeper_run";

ALTER TABLE "aux_housekeeper_run"
  DROP COLUMN "model_name",
  DROP COLUMN "count",
  ADD COLUMN "is_ok" boolean NOT NULL;

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
