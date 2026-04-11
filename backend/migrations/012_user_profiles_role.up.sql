-- 012_user_profiles_role.up.sql
-- Add role column to user_profiles

ALTER TABLE user_profiles ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user';
