-- 013_user_profiles_email.up.sql
-- Persist auth email on user_profiles for admin user listing and profile APIs

ALTER TABLE user_profiles
    ADD COLUMN IF NOT EXISTS email VARCHAR(320);

CREATE INDEX IF NOT EXISTS idx_user_profiles_email ON user_profiles(email);
