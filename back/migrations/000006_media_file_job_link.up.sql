-- A render belongs to the processing job that produced it. This also lets
-- multiple template renders of one clip coexist without metadata collisions.
ALTER TABLE media_files
  ADD COLUMN processing_job_id UUID REFERENCES processing_jobs(id) ON DELETE CASCADE;

CREATE INDEX media_files_processing_job_idx ON media_files(processing_job_id);
