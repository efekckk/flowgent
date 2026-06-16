CREATE TABLE triggers (
  id text PRIMARY KEY,
  workflow_id text NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  kind text NOT NULL CHECK (kind IN ('cron', 'webhook')),
  config jsonb NOT NULL DEFAULT '{}'::jsonb,
  enabled boolean NOT NULL DEFAULT true,
  last_fired_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX triggers_workflow_idx ON triggers (workflow_id);
CREATE INDEX triggers_enabled_kind_idx ON triggers (enabled, kind) WHERE enabled;

CREATE TRIGGER triggers_set_updated_at
BEFORE UPDATE ON triggers
FOR EACH ROW EXECUTE FUNCTION storage_set_updated_at();
