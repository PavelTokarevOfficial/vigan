CREATE UNIQUE INDEX processing_jobs_active_unique
  ON processing_jobs(clip_id, type)
  WHERE status IN ('pending', 'running');
