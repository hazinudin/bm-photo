-- Add upload_status and retry_count columns to photos table for retry support
ALTER TABLE photos ADD COLUMN IF NOT EXISTS upload_status VARCHAR(20) NOT NULL DEFAULT 'pending' 
    CHECK (upload_status IN ('pending', 'completed', 'expired'));

ALTER TABLE photos ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0 
    CHECK (retry_count >= 0 AND retry_count <= 5);

-- Create index for upload status queries
CREATE INDEX IF NOT EXISTS idx_photos_upload_status ON photos(upload_status) WHERE deleted_at IS NULL;

-- Note: This migration assumes migration 000002 has been applied and the 'status' column was removed.
-- The new upload_status column serves a different purpose - tracking the upload lifecycle specifically.
