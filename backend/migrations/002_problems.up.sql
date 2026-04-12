-- 002_problems.sql
-- DOMjudge-style problem schema

-- Problems table
CREATE TABLE problems (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,

    -- Time limits (seconds - DOMjudge style)
    time_limit DECIMAL(10, 3) NOT NULL DEFAULT 1.0,

    -- Resource limits
    memory_limit INTEGER DEFAULT 524288,    -- kilobytes (512 MB)
    output_limit INTEGER DEFAULT 4096,      -- kilobytes (4 MB)
    process_limit INTEGER DEFAULT 50,

    -- Special judging
    special_run_id UUID,
    special_compare_id UUID,
    special_compare_args TEXT,

    -- Problem statement
    problem_statement_path TEXT,

    -- Metadata
    difficulty VARCHAR(20) CHECK (difficulty IN ('easy', 'medium', 'hard')),
    color VARCHAR(20),
    points INTEGER DEFAULT 1,

    -- Visibility
    allow_submit BOOLEAN DEFAULT true,
    allow_judge BOOLEAN DEFAULT true,
    is_published BOOLEAN DEFAULT false,

    author_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_problems_published ON problems(is_published);
CREATE INDEX idx_problems_special_run ON problems(special_run_id);
CREATE INDEX idx_problems_special_compare ON problems(special_compare_id);

-- Test cases table
CREATE TABLE test_cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    problem_id UUID REFERENCES problems(id) ON DELETE CASCADE,

    -- Ordering and visibility
    rank INTEGER NOT NULL DEFAULT 0,
    is_sample BOOLEAN DEFAULT false,

    -- File references (object storage)
    input_path TEXT NOT NULL,
    output_path TEXT NOT NULL,

    -- Checksums for caching
    md5sum_input CHAR(32),
    md5sum_output CHAR(32),

    -- Original filenames
    orig_input_filename VARCHAR(255),
    orig_output_filename VARCHAR(255),

    -- Metadata
    description TEXT,
    is_interactive BOOLEAN DEFAULT false,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(problem_id, rank)
);

CREATE INDEX idx_test_cases_problem ON test_cases(problem_id);
CREATE INDEX idx_test_cases_sample ON test_cases(is_sample);

-- Executables table (validators, run scripts)
CREATE TABLE executables (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(190) UNIQUE NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('run', 'compare', 'compile')),
    description TEXT,

    executable_path TEXT NOT NULL,
    md5sum CHAR(32),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_executables_type ON executables(type);

-- Languages table
CREATE TABLE languages (
    id VARCHAR(50) PRIMARY KEY,
    external_id VARCHAR(50) UNIQUE,
    name VARCHAR(255) NOT NULL,

    compile_script_id UUID REFERENCES executables(id),

    time_factor DECIMAL(10, 3) DEFAULT 1.0,
    extensions TEXT[] NOT NULL,
    filter_compiler_warnings TEXT,

    allow_submit BOOLEAN DEFAULT true,
    allow_judge BOOLEAN DEFAULT true,

    require_entry_point BOOLEAN DEFAULT false,
    entry_point_description TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);