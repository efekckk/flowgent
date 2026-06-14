CREATE EXTENSION IF NOT EXISTS citext;

CREATE OR REPLACE FUNCTION storage_set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;

CREATE TABLE users (
  id             text PRIMARY KEY,
  email          citext UNIQUE NOT NULL,
  password_hash  text NOT NULL,
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION storage_set_updated_at();

CREATE INDEX users_created_at_idx ON users (created_at);
