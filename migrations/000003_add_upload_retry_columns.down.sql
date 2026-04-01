-- Revert: Remove upload_status and retry_count columns from photos table
DROP INDEX IF EXISTS idx_photos_upload_status;

ALTER TABLE photos DROP COLUMN IF EXISTS retry_count;
ALTER TABLE photos DROP COLUMN IF EXISTS upload_status;

-- Note: This reverts to the state after migration 000002 (simplified schema without upload tracking columns)
