CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE streamers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  twitch_login TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  twitch_user_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TYPE clip_status AS ENUM ('saved','downloading','downloaded','transcribing','ready_to_render','rendering','completed','failed');
CREATE TABLE clips (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  streamer_id UUID NOT NULL REFERENCES streamers(id) ON DELETE CASCADE,
  twitch_clip_id TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  twitch_url TEXT NOT NULL,
  thumbnail_url TEXT,
  duration DOUBLE PRECISION,
  twitch_created_at TIMESTAMPTZ,
  status clip_status NOT NULL DEFAULT 'saved',
  error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TYPE media_type AS ENUM ('source','audio','subtitle','render','banner');
CREATE TABLE media_files (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clip_id UUID REFERENCES clips(id) ON DELETE CASCADE,
  banner_id UUID,
  type media_type NOT NULL,
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  size BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE banners (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  storage_key TEXT NOT NULL UNIQUE,
  mime_type TEXT NOT NULL,
  size BIGINT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE media_files ADD CONSTRAINT media_files_banner_id_fkey FOREIGN KEY (banner_id) REFERENCES banners(id) ON DELETE CASCADE;
CREATE TYPE job_type AS ENUM ('download','process');
CREATE TYPE job_status AS ENUM ('pending','running','completed','failed');
CREATE TABLE processing_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clip_id UUID NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
  banner_id UUID REFERENCES banners(id) ON DELETE SET NULL,
  type job_type NOT NULL,
  status job_status NOT NULL DEFAULT 'pending',
  current_step TEXT,
  progress INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  attempts INTEGER NOT NULL DEFAULT 0,
  error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX processing_jobs_claim_idx ON processing_jobs(status, created_at) WHERE status = 'pending';
CREATE INDEX clips_streamer_idx ON clips(streamer_id, created_at DESC);
