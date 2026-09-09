DROP INDEX IF EXISTS media_files_processing_job_idx;
ALTER TABLE media_files DROP COLUMN IF EXISTS processing_job_id;
