-- 008_languages_compile.up.sql
-- Add compile commands and related configuration to languages table

ALTER TABLE languages ADD COLUMN IF NOT EXISTS compile_command TEXT;
ALTER TABLE languages ADD COLUMN IF NOT EXISTS run_command TEXT;
ALTER TABLE languages ADD COLUMN IF NOT EXISTS version VARCHAR(50);