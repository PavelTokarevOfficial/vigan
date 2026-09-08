UPDATE clips c
SET status = 'downloaded', updated_at = now()
WHERE c.status = 'completed'
  AND EXISTS (
    SELECT 1 FROM processing_jobs j
    WHERE j.clip_id = c.id AND j.type = 'download' AND j.status = 'completed'
  )
  AND NOT EXISTS (
    SELECT 1 FROM processing_jobs j
    WHERE j.clip_id = c.id AND j.type = 'process'
  );
