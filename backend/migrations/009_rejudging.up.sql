-- 009_rejudging.up.sql
-- Rejudging system for selective rejudging with apply/revert support

-- rejudgings: tracks rejudging operations
CREATE TABLE rejudgings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,            -- admin who created it
    contest_id UUID,                  -- optional, for contest rejudges

    -- Filters for what to rejudge
    problem_id UUID,                  -- optional filter
    submission_ids UUID[] DEFAULT '{}', -- specific submissions (if empty, use filters)
    from_verdict VARCHAR(20),         -- optional filter by original verdict

    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN (
        'pending', 'judging', 'judged', 'applied', 'reverted', 'cancelled'
    )),
    reason TEXT,                      -- why this rejudge was requested

    -- Result tracking
    affected_count INTEGER DEFAULT 0, -- number of submissions affected

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    applied_at TIMESTAMP WITH TIME ZONE,
    reverted_at TIMESTAMP WITH TIME ZONE
);

-- rejudging_submissions: tracks individual submissions in a rejudge
CREATE TABLE rejudging_submissions (
    rejudging_id UUID REFERENCES rejudgings(id) ON DELETE CASCADE,
    submission_id UUID REFERENCES submissions(id),
    original_judging_id UUID REFERENCES judgings(id),
    new_judging_id UUID REFERENCES judgings(id),

    original_verdict VARCHAR(20),
    new_verdict VARCHAR(20),
    verdict_changed BOOLEAN DEFAULT false,

    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN (
        'pending', 'judging', 'done'
    )),

    PRIMARY KEY (rejudging_id, submission_id)
);

-- Indexes
CREATE INDEX idx_rejudgings_status ON rejudgings(status);
CREATE INDEX idx_rejudgings_user ON rejudgings(user_id);
CREATE INDEX idx_rejudgings_contest ON rejudgings(contest_id);
CREATE INDEX idx_rejudgings_problem ON rejudgings(problem_id);
CREATE INDEX idx_rejudgings_created ON rejudgings(created_at DESC);
CREATE INDEX idx_rejudging_submissions_rejudging ON rejudging_submissions(rejudging_id);
CREATE INDEX idx_rejudging_submissions_submission ON rejudging_submissions(submission_id);
CREATE INDEX idx_rejudging_submissions_status ON rejudging_submissions(status);

-- Add rejudge_id to judgings table to link judgings to rejudging operations
ALTER TABLE judgings ADD COLUMN rejudging_id UUID REFERENCES rejudgings(id);
CREATE INDEX idx_judgings_rejudging ON judgings(rejudging_id);