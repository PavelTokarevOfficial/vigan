UPDATE processing_jobs SET banner_id = NULL WHERE banner_id IS NOT NULL;

ALTER TABLE processing_jobs DROP COLUMN banner_id;
ALTER TABLE media_files DROP COLUMN banner_id;
DROP TABLE banners;

CREATE TYPE media_type_without_banner AS ENUM ('source', 'audio', 'subtitle', 'render');
ALTER TABLE media_files ALTER COLUMN type TYPE media_type_without_banner USING type::text::media_type_without_banner;
DROP TYPE media_type;
ALTER TYPE media_type_without_banner RENAME TO media_type;
