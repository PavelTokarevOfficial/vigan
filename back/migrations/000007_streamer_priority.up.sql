ALTER TABLE streamers
  ADD COLUMN priority INTEGER NOT NULL DEFAULT 0 CHECK (priority >= 0);

CREATE INDEX streamers_priority_idx ON streamers(priority DESC, created_at DESC);
