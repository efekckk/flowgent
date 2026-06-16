CREATE TABLE credentials (
  id              text PRIMARY KEY,
  workspace_id    text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name            text NOT NULL,
  type            text NOT NULL,
  encrypted       bytea NOT NULL,
  meta            jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, name)
);

CREATE TRIGGER credentials_set_updated_at
BEFORE UPDATE ON credentials
FOR EACH ROW EXECUTE FUNCTION storage_set_updated_at();

CREATE INDEX credentials_workspace_type_idx ON credentials (workspace_id, type);
