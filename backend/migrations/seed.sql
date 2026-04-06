-- Seed data for testing
-- This file provides SQL-based seeding as a fallback option
-- The primary seeding mechanism is the Go seed script: backend/cmd/seed/main.go
-- Run with: make seed (Go script) or migrate with this file included

-- Insert languages (with compile/run commands)
INSERT INTO languages (id, external_id, name, time_factor, extensions, allow_submit, allow_judge, compile_command, run_command, version) VALUES
('cpp', 'cpp', 'C++17', 1.0, ARRAY['.cpp', '.cc', '.cxx'], true, true, 'g++ -std=c++17 -O2 -o {executable} {source}', './{executable}', 'g++ (GCC) 13'),
('c', 'c', 'C11', 1.0, ARRAY['.c'], true, true, 'gcc -std=c11 -O2 -o {executable} {source}', './{executable}', 'gcc (GCC) 13'),
('python3', 'python3', 'Python 3', 2.0, ARRAY['.py', '.py3'], true, true, NULL, 'python3 {source}', 'Python 3.12'),
('java', 'java', 'Java 17', 1.5, ARRAY['.java'], true, true, 'javac -J-Xmx1024m -source 17 -target 17 {source}', 'java -Xmx512m {mainclass}', 'OpenJDK 17'),
('go', 'go', 'Go 1.21', 1.2, ARRAY['.go'], true, true, 'go build -o {executable} {source}', './{executable}', 'Go 1.21'),
('rust', 'rust', 'Rust', 1.0, ARRAY['.rs'], true, true, 'rustc -O -o {executable} {source}', './{executable}', 'Rust 1.75'),
('nodejs', 'nodejs', 'Node.js 18', 2.0, ARRAY['.js', '.mjs'], true, true, NULL, 'node {source}', 'Node.js 18.19')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    time_factor = EXCLUDED.time_factor,
    extensions = EXCLUDED.extensions,
    compile_command = EXCLUDED.compile_command,
    run_command = EXCLUDED.run_command,
    version = EXCLUDED.version;

-- Sample problem: A + B (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('A', 'A + B', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

-- Sample test cases for A + B
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/A/input/sample1.txt', 'problems/A/output/sample1.txt', '1 2\n', '3\n', 'Simple positive numbers'
FROM problems WHERE external_id = 'A'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 2, true, 'problems/A/input/sample2.txt', 'problems/A/output/sample2.txt', '-5 10\n', '5\n', 'Mixed positive and negative'
FROM problems WHERE external_id = 'A'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Sample problem: Sum of Array (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('B', 'Sum of Array', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

-- Sample test cases for Sum of Array
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/B/input/sample1.txt', 'problems/B/output/sample1.txt', '5\n1 2 3 4 5\n', '15\n', 'Simple positive numbers'
FROM problems WHERE external_id = 'B'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Sample problem: Fibonacci Number (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('C', 'Fibonacci Number', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

-- Sample test cases for Fibonacci Number
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/C/input/sample1.txt', 'problems/C/output/sample1.txt', '0\n', '0\n', 'Base case F(0)'
FROM problems WHERE external_id = 'C'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 2, true, 'problems/C/input/sample2.txt', 'problems/C/output/sample2.txt', '10\n', '55\n', 'F(10)'
FROM problems WHERE external_id = 'C'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Sample problem: Binary Search (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('D', 'Binary Search', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

-- Sample test cases for Binary Search
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/D/input/sample1.txt', 'problems/D/output/sample1.txt', '5\n1 3 5 7 9\n5\n', '2\n', 'Found in middle'
FROM problems WHERE external_id = 'D'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 2, true, 'problems/D/input/sample2.txt', 'problems/D/output/sample2.txt', '5\n1 3 5 7 9\n4\n', '-1\n', 'Not found'
FROM problems WHERE external_id = 'D'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Sample problem: Two Sum (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('E', 'Two Sum', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

-- Sample test cases for Two Sum
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/E/input/sample1.txt', 'problems/E/output/sample1.txt', '4\n2 7 11 15\n9\n', '0 1\n', 'First two elements'
FROM problems WHERE external_id = 'E'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Sample problem: LCS (Hard)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('F', 'Longest Common Subsequence', 2.0, 524288, 'hard', 300, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

-- Sample test cases for LCS
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/F/input/sample1.txt', 'problems/F/output/sample1.txt', 'abcde\nace\n', '3\n', 'LCS ace'
FROM problems WHERE external_id = 'F'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Sample problem: Shortest Path (Hard)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('G', 'Shortest Path', 2.0, 262144, 'hard', 300, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

-- Sample test cases for Shortest Path
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/G/input/sample1.txt', 'problems/G/output/sample1.txt', '4 4\n1 2 1\n2 3 2\n3 4 3\n1 4 10\n', '6\n', 'Path 1-2-3-4'
FROM problems WHERE external_id = 'G'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Sample contest
INSERT INTO contests (external_id, name, short_name, start_time, end_time, public) VALUES
('intro-contest-2024', 'Introduction to Algorithms Contest', 'INTRO2024', NOW() - INTERVAL '1 day', NOW() + INTERVAL '7 days', true)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

-- Add problems to contest
INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'A', 1, 100, true
FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'A'
ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'B', 2, 100, true
FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'B'
ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'C', 3, 100, true
FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'C'
ON CONFLICT DO NOTHING;