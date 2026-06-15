CREATE TABLE workflow_runs (
  id                text PRIMARY KEY,
  workflow_id       text NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  workflow_version  int  NOT NULL,
  status            text NOT NULL
                      CHECK (status IN ('queued','running','succeeded','failed','cancelled')),
  trigger_kind      text NOT NULL
                      CHECK (trigger_kind IN ('manual','webhook','cron')),
  trigger_payload   jsonb,
  error             text,
  started_at        timestamptz,
  finished_at       timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX workflow_runs_workflow_created_idx ON workflow_runs (workflow_id, created_at DESC);
CREATE INDEX workflow_runs_active_idx ON workflow_runs (status) WHERE status IN ('queued','running');

CREATE TABLE node_runs (
  id               text PRIMARY KEY,
  workflow_run_id  text NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
  node_id          text NOT NULL,
  iteration        int  NOT NULL DEFAULT 0,
  status           text NOT NULL
                     CHECK (status IN ('pending','running','succeeded','failed','skipped')),
  input            jsonb,
  output           jsonb,
  error            jsonb,
  attempts         int  NOT NULL DEFAULT 0,
  started_at       timestamptz,
  finished_at      timestamptz,
  duration_ms      int
);

CREATE INDEX node_runs_run_node_iter_idx ON node_runs (workflow_run_id, node_id, iteration);
