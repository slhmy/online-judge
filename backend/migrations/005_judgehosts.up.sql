-- 005_judgehosts.sql
-- Judgehost and rejudging schema

-- Judgehosts table
CREATE TABLE judgehosts (
    id VARCHAR(100) PRIMARY KEY,

    active BOOLEAN DEFAULT true,
    status VARCHAR(20) DEFAULT 'idle' CHECK (status IN ('idle', 'judging', 'error', 'disabled')),

    last_ping TIMESTAMP WITH TIME ZONE,
    last_error TEXT,

    polltime INTEGER DEFAULT 10,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Rejudgings table
CREATE TABLE rejudgings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    reason TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'cancelled')),

    problem_id UUID REFERENCES problems(id),
    language_id VARCHAR(50) REFERENCES languages(id),
    user_id UUID,
    contest_id UUID REFERENCES contests(id),

    total_submissions INTEGER DEFAULT 0,
    judged_count INTEGER DEFAULT 0,

    applied_count INTEGER DEFAULT 0,
    cancelled_count INTEGER DEFAULT 0,

    auto_apply BOOLEAN DEFAULT false,
    auto_apply_at TIMESTAMP WITH TIME ZONE,

    created_by VARCHAR(100) NOT NULL,

    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_rejudgings_status ON rejudgings(status);
CREATE INDEX idx_rejudgings_problem ON rejudgings(problem_id);