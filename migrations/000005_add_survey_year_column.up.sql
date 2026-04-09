-- Add survey_year column to photos table
-- This column tracks the year when the survey was conducted, extracted from uploaded_at

-- Add column if not exists (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'photos' AND column_name = 'survey_year') THEN
        ALTER TABLE photos ADD COLUMN survey_year INTEGER NOT NULL DEFAULT EXTRACT(YEAR FROM CURRENT_TIMESTAMP)::INTEGER;
    END IF;
END $$;

-- Backfill existing rows with year from uploaded_at
UPDATE photos SET survey_year = EXTRACT(YEAR FROM uploaded_at)::INTEGER WHERE survey_year IS NULL;

-- Index for efficient queries filtering by survey year and route (idempotent)
CREATE INDEX IF NOT EXISTS idx_photos_survey_year ON photos(survey_year, route_id) WHERE deleted_at IS NULL;

COMMENT ON COLUMN photos.survey_year IS 'Year when the survey was conducted, extracted from uploaded_at timestamp';