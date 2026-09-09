DROP INDEX IF EXISTS streamers_priority_idx;

ALTER TABLE streamers DROP COLUMN IF EXISTS priority;
