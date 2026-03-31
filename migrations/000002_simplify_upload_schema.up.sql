-- Simplify photos table: remove status, thumbnails, and EXIF columns
ALTER TABLE photos DROP COLUMN IF EXISTS status;
ALTER TABLE photos DROP COLUMN IF EXISTS thumbnail_small_path;
ALTER TABLE photos DROP COLUMN IF EXISTS thumbnail_medium_path;
ALTER TABLE photos DROP COLUMN IF EXISTS thumbnail_large_path;
ALTER TABLE photos DROP COLUMN IF EXISTS exif_data;
ALTER TABLE photos DROP COLUMN IF EXISTS processing_completed_at;

-- Make sta_value and sta_source nullable
ALTER TABLE photos ALTER COLUMN sta_value DROP NOT NULL;
ALTER TABLE photos ALTER COLUMN sta_source DROP NOT NULL;

-- Simplify pending_uploads table: remove file tracking columns
ALTER TABLE pending_uploads DROP COLUMN IF EXISTS file_name;
ALTER TABLE pending_uploads DROP COLUMN IF EXISTS content_type;
ALTER TABLE pending_uploads DROP COLUMN IF EXISTS file_size_bytes;
ALTER TABLE pending_uploads DROP COLUMN IF EXISTS completed_at;

-- Update indexes (status-related indexes no longer needed)
DROP INDEX IF EXISTS idx_photos_status;
DROP INDEX IF EXISTS idx_pending_uploads_api_key;

-- Create updated index for pending uploads (only pending status)
CREATE INDEX idx_pending_uploads_api_key ON pending_uploads(api_key_id) WHERE status = 'pending';
