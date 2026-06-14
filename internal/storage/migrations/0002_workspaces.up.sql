CREATE TABLE workspaces (
  id             text PRIMARY KEY,
  owner_user_id  text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name           text NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX workspaces_owner_idx ON workspaces (owner_user_id);
