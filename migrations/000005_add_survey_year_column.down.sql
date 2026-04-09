-- Remove survey_year column and related index from photos table

DROP INDEX IF EXISTS idx_photos_survey_year;

ALTER TABLE photos DROP COLUMN IF EXISTS survey_year;