CREATE TABLE workflows (
  id                          text PRIMARY KEY,
  workspace_id                text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  name                        text NOT NULL,
  status                      text NOT NULL
                                CHECK (status IN ('draft','active','paused','archived')),
  current_version             int  NOT NULL DEFAULT 1,
  default_llm_credential_id   text,
  created_at                  timestamptz NOT NULL DEFAULT now(),
  updated_at                  timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER workflows_set_updated_at
BEFORE UPDATE ON workflows
FOR EACH ROW EXECUTE FUNCTION storage_set_updated_at();

CREATE INDEX workflows_workspace_status_idx ON workflows (workspace_id, status);

CREATE TABLE workflow_versions (
  id                  text PRIMARY KEY,
  workflow_id         text NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  version             int  NOT NULL,
  definition          jsonb NOT NULL,
  message             text,
  created_by_user_id  text REFERENCES users(id) ON DELETE SET NULL,
  created_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (workflow_id, version)
);

CREATE INDEX workflow_versions_workflow_idx ON workflow_versions (workflow_id, version DESC);
