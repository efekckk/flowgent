DROP TABLE IF EXISTS run_logs CASCADE;

DROP INDEX IF EXISTS workflow_runs_trigger_idx;
DROP INDEX IF EXISTS workflow_runs_parent_idx;
DROP INDEX IF EXISTS workflow_runs_status_idx;

ALTER TABLE workflow_runs DROP CONSTRAINT IF EXISTS workflow_runs_trigger_kind_check;
ALTER TABLE workflow_runs ADD CONSTRAINT workflow_runs_trigger_kind_check
  CHECK (trigger_kind IN ('manual','webhook','cron'));

ALTER TABLE workflow_runs ALTER COLUMN trigger_payload DROP NOT NULL;
ALTER TABLE workflow_runs ALTER COLUMN trigger_payload DROP DEFAULT;

ALTER TABLE workflow_runs DROP COLUMN IF EXISTS parent_run_id;
ALTER TABLE workflow_runs DROP COLUMN IF EXISTS trigger_id;
