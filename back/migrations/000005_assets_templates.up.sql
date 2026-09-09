CREATE TYPE asset_kind AS ENUM ('image', 'gif', 'video', 'audio');

CREATE TABLE asset_folders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_id UUID REFERENCES asset_folders(id) ON DELETE RESTRICT,
  name TEXT NOT NULL CHECK (length(trim(name)) > 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX asset_folders_unique_name_per_parent
  ON asset_folders (COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(name));

CREATE TABLE assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  folder_id UUID REFERENCES asset_folders(id) ON DELETE RESTRICT,
  name TEXT NOT NULL CHECK (length(trim(name)) > 0),
  kind asset_kind NOT NULL,
  mime_type TEXT NOT NULL,
  size BIGINT NOT NULL CHECK (size >= 0),
  storage_key TEXT NOT NULL UNIQUE,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX assets_unique_name_per_folder
  ON assets (COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(name));

CREATE TABLE video_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL UNIQUE CHECK (length(trim(name)) > 0),
  description TEXT NOT NULL DEFAULT '',
  preview_asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
  config_version INTEGER NOT NULL DEFAULT 1 CHECK (config_version > 0),
  config JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE template_asset_references (
  template_id UUID NOT NULL REFERENCES video_templates(id) ON DELETE CASCADE,
  asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
  PRIMARY KEY (template_id, asset_id)
);

ALTER TABLE processing_jobs
  ADD COLUMN template_id UUID REFERENCES video_templates(id) ON DELETE SET NULL,
  ADD COLUMN template_snapshot JSONB;
CREATE INDEX processing_jobs_template_idx ON processing_jobs(template_id);

-- The default preserves the former hard-coded vertical composition. New jobs save
-- an immutable copy of this config (and any referenced asset storage keys).
INSERT INTO video_templates(name, description, config_version, config) VALUES (
  'Классический vertical',
  'Размытый фон, исходный клип по центру и субтитры снизу.',
  1,
  '{
    "version": 1,
    "canvas": {"width": 1080, "height": 1920, "fps": 30, "background": "#000000"},
    "layers": [
      {"id": "background", "name": "Размытый фон", "type": "input_video", "x": 0, "y": 0, "width": 1080, "height": 1920, "visible": true, "opacity": 1, "fit": "cover", "filters": {"blur": 25, "brightness": -0.2}},
      {"id": "clip", "name": "Основной клип", "type": "input_video", "x": 0, "y": 0, "width": 1080, "height": 1920, "visible": true, "opacity": 1, "fit": "contain"},
      {"id": "subtitles", "name": "Субтитры", "type": "subtitles", "x": 90, "y": 1540, "width": 900, "height": 240, "visible": true, "opacity": 1, "style": {"fontSize": 8, "alignment": 2, "marginV": 100, "outline": 2, "primaryColor": "&H00FFFFFF", "outlineColor": "&H00000000"}}
    ]
  }'::jsonb
);
