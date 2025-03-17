DROP FUNCTION housekeeper_ran_in_last;
DELETE FROM "aux_housekeeper_run";

ALTER TABLE "aux_housekeeper_run"
  ADD COLUMN "model_name" varchar NOT NULL,
  ADD COLUMN "count" bigint NOT NULL,
  DROP COLUMN "is_ok";

-- housekeeper_ran_in_last function returns TRUE when the housekeeper has
-- successfully processed stale records for a given model name in the last VAL
-- interval.
CREATE FUNCTION housekeeper_ran_in_last(val interval, model text)
RETURNS BOOLEAN AS
$func$
       SELECT EXISTS(
           SELECT
               id
           FROM aux_housekeeper_run
           WHERE model_name = model AND completed_at > NOW() - val
       );
$func$ LANGUAGE SQL STABLE;
