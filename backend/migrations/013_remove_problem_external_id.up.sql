-- Remove deprecated problem external ID field.
DROP INDEX IF EXISTS idx_problems_external_id;
ALTER TABLE problems DROP COLUMN IF EXISTS external_id;
