-- 004_contests.sql
-- DOMjudge-style contest schema

-- Contests table
CREATE TABLE contests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(190) UNIQUE,

    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),

    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    freeze_time TIMESTAMP WITH TIME ZONE,
    unfreeze_time TIMESTAMP WITH TIME ZONE,
    finalize_time TIMESTAMP WITH TIME ZONE,

    activation_time TIMESTAMP WITH TIME ZONE,
    public BOOLEAN DEFAULT true,

    contest_file_path TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_contests_time ON contests(start_time, end_time);
CREATE INDEX idx_contests_public ON contests(public);

-- Contest problems table
CREATE TABLE contest_problems (
    contest_id UUID REFERENCES contests(id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(id),

    short_name VARCHAR(10) NOT NULL,
    rank INTEGER NOT NULL,
    color VARCHAR(20),

    points INTEGER DEFAULT 1,

    allow_submit BOOLEAN DEFAULT true,
    allow_judge BOOLEAN DEFAULT true,

    PRIMARY KEY (contest_id, problem_id)
);

CREATE INDEX idx_contest_problems_contest ON contest_problems(contest_id);

-- Teams table
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE REFERENCES user_profiles(user_id),

    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),

    affiliation VARCHAR(255),
    country VARCHAR(3),

    contest_id UUID REFERENCES contests(id),

    points INTEGER DEFAULT 0,
    total_time INTEGER DEFAULT 0,

    room VARCHAR(255),
    location VARCHAR(255),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_teams_contest ON teams(contest_id);
CREATE INDEX idx_teams_user ON teams(user_id);

-- Contest participants
CREATE TABLE contest_participants (
    contest_id UUID REFERENCES contests(id) ON DELETE CASCADE,
    user_id UUID REFERENCES user_profiles(user_id),
    registered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    PRIMARY KEY (contest_id, user_id)
);