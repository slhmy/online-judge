-- 010_judgehosts_capabilities.up.sql
-- Add capabilities and queue management columns to judgehosts

ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS queue_name VARCHAR(100) DEFAULT '';
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS languages TEXT[] DEFAULT '{}';
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS max_concurrent INTEGER DEFAULT 5;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS memory_limit BIGINT DEFAULT 0;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS time_limit DOUBLE PRECISION DEFAULT 0;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS supports_interactive BOOLEAN DEFAULT false;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS supports_special BOOLEAN DEFAULT false;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS extra JSONB DEFAULT '{}'::jsonb;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS current_job_id VARCHAR(100) DEFAULT '';
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS active_jobs INTEGER DEFAULT 0;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS completed_jobs INTEGER DEFAULT 0;
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS registered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Add constraint to ensure max_concurrent is positive
ALTER TABLE judgehosts ADD CONSTRAINT chk_max_concurrent_positive CHECK (max_concurrent > 0);

-- Create index for quick lookup of available judgehosts
CREATE INDEX IF NOT EXISTS idx_judgehosts_status ON judgehosts(status);
CREATE INDEX IF NOT EXISTS idx_judgehosts_last_ping ON judgehosts(last_ping DESC);
CREATE INDEX IF NOT EXISTS idx_judgehosts_languages ON judgehosts USING GIN(languages);
CREATE INDEX IF NOT EXISTS idx_judgehosts_queue_name ON judgehosts(queue_name);

-- Update status constraint to include 'busy' and 'offline'
ALTER TABLE judgehosts DROP CONSTRAINT IF EXISTS judgehosts_status_check;
ALTER TABLE judgehosts ADD CONSTRAINT judgehosts_status_check
    CHECK (status IN ('idle', 'busy', 'judging', 'offline', 'error', 'disabled', 'unspecified'));