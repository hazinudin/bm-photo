-- Revert photos table changes: add back status, thumbnails, and EXIF columns
ALTER TABLE photos ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'processing';
ALTER TABLE photos ADD COLUMN thumbnail_small_path VARCHAR(500);
ALTER TABLE photos ADD COLUMN thumbnail_medium_path VARCHAR(500);
ALTER TABLE photos ADD COLUMN thumbnail_large_path VARCHAR(500);
ALTER TABLE photos ADD COLUMN exif_data JSONB;
ALTER TABLE photos ADD COLUMN processing_completed_at TIMESTAMP WITH TIME ZONE;

-- Make sta_value and sta_source NOT NULL again
ALTER TABLE photos ALTER COLUMN sta_value SET NOT NULL;
ALTER TABLE photos ALTER COLUMN sta_source SET NOT NULL;

-- Revert pending_uploads table: add back file tracking columns
ALTER TABLE pending_uploads ADD COLUMN file_name VARCHAR(255) NOT NULL DEFAULT 'unknown';
ALTER TABLE pending_uploads ADD COLUMN content_type VARCHAR(100) NOT NULL DEFAULT 'application/octet-stream';
ALTER TABLE pending_uploads ADD COLUMN file_size_bytes BIGINT NOT NULL DEFAULT 0;
ALTER TABLE pending_uploads ADD COLUMN completed_at TIMESTAMP WITH TIME ZONE;

-- Re-create indexes
CREATE INDEX idx_photos_status ON photos(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_pending_uploads_api_key ON pending_uploads(api_key_id) WHERE status IN ('pending', 'uploaded');
