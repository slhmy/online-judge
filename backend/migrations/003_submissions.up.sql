-- 003_submissions.sql
-- DOMjudge-style submission and judging schema

-- Submissions table
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL,
    problem_id UUID REFERENCES problems(id),
    contest_id UUID,
    language_id VARCHAR(50) REFERENCES languages(id),

    source_path TEXT NOT NULL,

    -- Team priority for fair scheduling
    team_pending_count INTEGER DEFAULT 0,

    submit_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    team_id UUID,
    orig_submit_time TIMESTAMP WITH TIME ZONE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_submissions_user ON submissions(user_id);
CREATE INDEX idx_submissions_problem ON submissions(problem_id);
CREATE INDEX idx_submissions_contest ON submissions(contest_id);
CREATE INDEX idx_submissions_time ON submissions(submit_time DESC);
CREATE INDEX idx_submissions_team ON submissions(team_id);

-- Judgings table
CREATE TABLE judgings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID REFERENCES submissions(id),

    judgehost_id VARCHAR(100) NOT NULL,

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    max_runtime DECIMAL(10, 3),
    max_memory INTEGER,

    -- Verdict (DOMjudge style)
    verdict VARCHAR(20) CHECK (verdict IN (
        'correct', 'wrong-answer', 'timelimit',
        'memory-limit', 'run-error', 'compiler-error',
        'output-limit', 'presentation'
    )),

    -- Compilation
    compile_success BOOLEAN DEFAULT false,
    compile_output_path TEXT,
    compile_metadata JSONB,

    -- Validity
    valid BOOLEAN DEFAULT true,

    -- Verification
    verified BOOLEAN DEFAULT false,
    verified_by VARCHAR(100),
    verify_comment TEXT,
    seen BOOLEAN DEFAULT false,

    -- For partial scoring
    judge_completely BOOLEAN DEFAULT false,
    score DECIMAL(10, 3),

    uuid UUID DEFAULT uuid_generate_v4(),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_judgings_submission ON judgings(submission_id);
CREATE INDEX idx_judgings_valid ON judgings(valid);
CREATE INDEX idx_judgings_verdict ON judgings(verdict);
CREATE INDEX idx_judgings_judgehost ON judgings(judgehost_id);

-- Judging runs table (per testcase results)
CREATE TABLE judging_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    judging_id UUID REFERENCES judgings(id) ON DELETE CASCADE,
    test_case_id UUID REFERENCES test_cases(id),

    rank INTEGER NOT NULL,

    runtime DECIMAL(10, 3),
    wall_time DECIMAL(10, 3),
    memory INTEGER,

    verdict VARCHAR(20) NOT NULL,

    output_run_path TEXT,
    output_diff_path TEXT,
    output_error_path TEXT,
    output_system TEXT,

    metadata JSONB,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_judging_runs_judging ON judging_runs(judging_id);
CREATE INDEX idx_judging_runs_testcase ON judging_runs(test_case_id);