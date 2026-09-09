ALTER TABLE processing_jobs
  DROP COLUMN IF EXISTS template_snapshot,
  DROP COLUMN IF EXISTS template_id;
DROP INDEX IF EXISTS processing_jobs_template_idx;

DROP TABLE IF EXISTS template_asset_references;
DROP TABLE IF EXISTS video_templates;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS asset_folders;
DROP TYPE IF EXISTS asset_kind;
