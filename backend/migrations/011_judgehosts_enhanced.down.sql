-- 011_judgehosts_enhanced.down.sql
-- Rollback enhanced judgehosts fields

-- Remove indexes
DROP INDEX IF EXISTS idx_judgehosts_current_judging;
DROP INDEX IF EXISTS idx_judgehosts_queue_name;
DROP INDEX IF EXISTS idx_judgehosts_last_ping;
DROP INDEX IF EXISTS idx_judgehosts_status;
DROP INDEX IF EXISTS idx_judgehosts_metrics;
DROP INDEX IF EXISTS idx_judgehosts_capabilities;

-- Rename back to current_job_id
ALTER TABLE judgehosts RENAME COLUMN current_judging_id TO current_job_id;

-- Remove columns
ALTER TABLE judgehosts DROP COLUMN IF EXISTS metrics;
ALTER TABLE judgehosts DROP COLUMN IF EXISTS capabilities;