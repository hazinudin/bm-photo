-- Add composite index for efficient filename lookups within a route
-- This index enables fast case-insensitive filename queries scoped to a route
CREATE INDEX IF NOT EXISTS idx_photos_route_filename_lower
  ON photos(route_id, LOWER(original_filename))
  WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_photos_route_filename_lower IS 'Index for case-insensitive filename search within a route';
