-- 008_testcase_content.up.sql
-- Add inline content columns for sample test cases

ALTER TABLE test_cases ADD COLUMN IF NOT EXISTS input_content TEXT;
ALTER TABLE test_cases ADD COLUMN IF NOT EXISTS output_content TEXT;

-- Add problem statement content table
CREATE TABLE IF NOT EXISTS problem_statements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    problem_id UUID REFERENCES problems(id) ON DELETE CASCADE,

    language VARCHAR(10) DEFAULT 'en',
    format VARCHAR(20) DEFAULT 'html',  -- html, markdown, pdf

    title VARCHAR(255),
    content TEXT NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(problem_id, language)
);

CREATE INDEX IF NOT EXISTS idx_problem_statements_problem ON problem_statements(problem_id);