-- 001_users.sql
-- User profiles (separate from Identra auth)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- User profiles table
CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE,      -- References Identra user_id
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(255),

    -- Stats
    rating INTEGER DEFAULT 0,
    solved_count INTEGER DEFAULT 0,
    submission_count INTEGER DEFAULT 0,

    -- Metadata
    avatar_url TEXT,
    bio TEXT,
    country VARCHAR(3),                 -- ISO 3166-1 alpha-3

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_profiles_user ON user_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_username ON user_profiles(username);
CREATE INDEX IF NOT EXISTS idx_user_profiles_rating ON user_profiles(rating DESC);