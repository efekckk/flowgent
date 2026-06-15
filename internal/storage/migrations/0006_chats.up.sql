CREATE TABLE chat_threads (
  id           text PRIMARY KEY,
  workflow_id  text NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
  user_id      text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX chat_threads_workflow_idx ON chat_threads (workflow_id);

CREATE TABLE chat_messages (
  id            text PRIMARY KEY,
  thread_id     text NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
  seq           bigserial,
  role          text NOT NULL
                  CHECK (role IN ('system','user','assistant','tool')),
  content       text,
  tool_calls    jsonb,
  tool_results  jsonb,
  model         text,
  usage         jsonb,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX chat_messages_thread_seq_idx ON chat_messages (thread_id, seq);
