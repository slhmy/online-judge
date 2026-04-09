-- 011_judgehosts_enhanced.up.sql
-- Add consolidated capabilities and metrics JSONB fields, rename current_job_id

-- Add capabilities JSONB field for supported languages and resource limits
-- Example: {"languages": ["cpp", "python", "go"], "memory_limit": 524288, "time_limit": 60.0, "supports_interactive": true}
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS capabilities JSONB DEFAULT '{}'::jsonb;

-- Add metrics JSONB field for runtime metrics
-- Example: {"cpu_load": 45.2, "memory_used": 2048, "memory_total": 8192, "queue_depth": 3}
ALTER TABLE judgehosts ADD COLUMN IF NOT EXISTS metrics JSONB DEFAULT '{}'::jsonb;

-- Rename current_job_id to current_judging_id for clarity
ALTER TABLE judgehosts RENAME COLUMN current_job_id TO current_judging_id;

-- Add GIN index for capabilities JSONB queries
CREATE INDEX IF NOT EXISTS idx_judgehosts_capabilities ON judgehosts USING GIN(capabilities);

-- Add GIN index for metrics JSONB queries
CREATE INDEX IF NOT EXISTS idx_judgehosts_metrics ON judgehosts USING GIN(metrics);

-- Ensure the status and last_ping indexes exist (may already exist from migration 010)
CREATE INDEX IF NOT EXISTS idx_judgehosts_status ON judgehosts(status);
CREATE INDEX IF NOT EXISTS idx_judgehosts_last_ping ON judgehosts(last_ping DESC);

-- Add index for queue_name lookups
CREATE INDEX IF NOT EXISTS idx_judgehosts_queue_name ON judgehosts(queue_name);

-- Add index for current_judging_id lookups
CREATE INDEX IF NOT EXISTS idx_judgehosts_current_judging ON judgehosts(current_judging_id);