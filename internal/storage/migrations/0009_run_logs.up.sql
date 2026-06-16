-- 0009: link runs to triggers + parent runs (for replay) and add a separate
-- per-run log stream with a full-text search index. The new workflow_runs
-- columns are nullable references so existing rows survive; trigger_payload
-- is tightened to NOT NULL DEFAULT to make downstream consumers simpler.
--
-- Note: workflow_runs_workflow_created_idx already exists from migration
-- 0005, so we do not recreate it here. The status index added below is
-- the un-filtered companion to the existing partial active-only index.

ALTER TABLE workflow_runs ADD COLUMN trigger_id text REFERENCES triggers(id) ON DELETE SET NULL;
ALTER TABLE workflow_runs ADD COLUMN parent_run_id text REFERENCES workflow_runs(id) ON DELETE SET NULL;

-- Widen trigger_kind to include 'replay', which RunFromReplay stamps onto
-- forked runs so the history viewer can distinguish them.
ALTER TABLE workflow_runs DROP CONSTRAINT workflow_runs_trigger_kind_check;
ALTER TABLE workflow_runs ADD CONSTRAINT workflow_runs_trigger_kind_check
  CHECK (trigger_kind IN ('manual','webhook','cron','replay'));

UPDATE workflow_runs SET trigger_payload = '{}'::jsonb WHERE trigger_payload IS NULL;
ALTER TABLE workflow_runs ALTER COLUMN trigger_payload SET DEFAULT '{}'::jsonb;
ALTER TABLE workflow_runs ALTER COLUMN trigger_payload SET NOT NULL;

CREATE INDEX workflow_runs_status_idx ON workflow_runs (status);
CREATE INDEX workflow_runs_parent_idx ON workflow_runs (parent_run_id) WHERE parent_run_id IS NOT NULL;
CREATE INDEX workflow_runs_trigger_idx ON workflow_runs (trigger_id) WHERE trigger_id IS NOT NULL;

CREATE TABLE run_logs (
  id         bigserial PRIMARY KEY,
  run_id     text       NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  node_id    text,
  level      text       NOT NULL CHECK (level IN ('debug','info','warn','error')),
  message    text       NOT NULL,
  at         timestamptz NOT NULL DEFAULT now(),
  search_doc tsvector   GENERATED ALWAYS AS (to_tsvector('english', message)) STORED
);

CREATE INDEX run_logs_run_at_idx   ON run_logs (run_id, at);
CREATE INDEX run_logs_search_gin   ON run_logs USING gin (search_doc);
