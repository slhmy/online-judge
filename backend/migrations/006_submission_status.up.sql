-- 006_submission_status.up.sql
-- Add status column to submissions table

ALTER TABLE submissions ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'queued' CHECK (status IN (
    'pending', 'queued', 'judging', 'completed', 'error'
));

CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);