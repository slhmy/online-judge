-- Add source_code column to submissions table
ALTER TABLE submissions ADD COLUMN IF NOT EXISTS source_code TEXT;

-- Update source_path for existing records
UPDATE submissions SET source_path = 'stored-in-db' WHERE source_path IS NULL OR source_path = '';