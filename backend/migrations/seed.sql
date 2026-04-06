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

-- Problem A: A + B (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('A', 'A + B', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/A/input/sample1.txt', 'problems/A/output/sample1.txt', '1 2\n', '3\n', 'Simple positive numbers'
FROM problems WHERE external_id = 'A'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem B: Sum of Array (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('B', 'Sum of Array', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/B/input/sample1.txt', 'problems/B/output/sample1.txt', '5\n1 2 3 4 5\n', '15\n', 'Simple positive numbers'
FROM problems WHERE external_id = 'B'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem C: Fibonacci Number (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('C', 'Fibonacci Number', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/C/input/sample1.txt', 'problems/C/output/sample1.txt', '0\n', '0\n', 'Base case F(0)'
FROM problems WHERE external_id = 'C'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem D: Binary Search (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('D', 'Binary Search', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/D/input/sample1.txt', 'problems/D/output/sample1.txt', '5\n1 3 5 7 9\n5\n', '2\n', 'Found in middle'
FROM problems WHERE external_id = 'D'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem E: Two Sum (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('E', 'Two Sum', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/E/input/sample1.txt', 'problems/E/output/sample1.txt', '4\n2 7 11 15\n9\n', '0 1\n', 'First two elements'
FROM problems WHERE external_id = 'E'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem F: LCS (Hard)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('F', 'Longest Common Subsequence', 2.0, 524288, 'hard', 300, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/F/input/sample1.txt', 'problems/F/output/sample1.txt', 'abcde\nace\n', '3\n', 'LCS ace'
FROM problems WHERE external_id = 'F'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem G: Shortest Path (Hard)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('G', 'Shortest Path', 2.0, 262144, 'hard', 300, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/G/input/sample1.txt', 'problems/G/output/sample1.txt', '4 4\n1 2 1\n2 3 2\n3 4 3\n1 4 10\n', '6\n', 'Path 1-2-3-4'
FROM problems WHERE external_id = 'G'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem H: Maximum Subarray (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('H', 'Maximum Subarray', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/H/input/sample1.txt', 'problems/H/output/sample1.txt', '9\n-2 1 -3 4 -1 2 1 -5 4\n', '6\n', 'Kadane'
FROM problems WHERE external_id = 'H'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem I: Palindrome Check (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('I', 'Palindrome Check', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/I/input/sample1.txt', 'problems/I/output/sample1.txt', 'racecar\n', 'true\n', 'Palindrome'
FROM problems WHERE external_id = 'I'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem J: Prime Factorization (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('J', 'Prime Factorization', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/J/input/sample1.txt', 'problems/J/output/sample1.txt', '12\n', '2 2\n3 1\n', 'Prime factors'
FROM problems WHERE external_id = 'J'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem K: Stack Implementation (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('K', 'Stack Implementation', 1.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/K/input/sample1.txt', 'problems/K/output/sample1.txt', '5\npush 10\ntop\n', '10\n', 'Basic test'
FROM problems WHERE external_id = 'K'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem L: String Matching (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('L', 'String Matching', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/L/input/sample1.txt', 'problems/L/output/sample1.txt', 'abab\nab\n', '0 2\n', 'Multiple matches'
FROM problems WHERE external_id = 'L'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem M: Sorting (Easy)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('M', 'Sorting Algorithms', 2.0, 262144, 'easy', 100, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/M/input/sample1.txt', 'problems/M/output/sample1.txt', '3\n3 1 2\n', '1 2 3\n', 'Basic sort'
FROM problems WHERE external_id = 'M'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem N: Tree Traversal (Hard)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('N', 'Tree Traversal', 2.0, 524288, 'hard', 300, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/N/input/sample1.txt', 'problems/N/output/sample1.txt', '3\n2 1 3\n1 2 3\n', '2 3 1\n', 'Simple tree'
FROM problems WHERE external_id = 'N'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Problem O: Graph DFS (Medium)
INSERT INTO problems (external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('O', 'Graph DFS', 1.0, 262144, 'medium', 200, true, true)
ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT id, 1, true, 'problems/O/input/sample1.txt', 'problems/O/output/sample1.txt', '3 2\n1 2\n2 3\n', '1 2 3\n', 'DFS traversal'
FROM problems WHERE external_id = 'O'
ON CONFLICT (problem_id, rank) DO UPDATE SET input_content = EXCLUDED.input_content;

-- Intro Contest
INSERT INTO contests (external_id, name, short_name, start_time, end_time, public) VALUES
('intro-contest-2024', 'Introduction to Algorithms Contest', 'INTRO2024', NOW() - INTERVAL '1 day', NOW() + INTERVAL '7 days', true)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'A', 1, 100, true FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'A' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'B', 2, 100, true FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'B' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'C', 3, 100, true FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'C' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'D', 4, 100, true FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'I' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'E', 5, 100, true FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'K' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'F', 6, 100, true FROM contests c, problems p
WHERE c.external_id = 'intro-contest-2024' AND p.external_id = 'M' ON CONFLICT DO NOTHING;

-- Advanced Contest
INSERT INTO contests (external_id, name, short_name, start_time, end_time, public) VALUES
('advanced-contest-2024', 'Advanced Algorithms Contest', 'ADV2024', NOW() - INTERVAL '2 hours', NOW() + INTERVAL '5 days', true)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'A', 1, 200, true FROM contests c, problems p
WHERE c.external_id = 'advanced-contest-2024' AND p.external_id = 'D' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'B', 2, 200, true FROM contests c, problems p
WHERE c.external_id = 'advanced-contest-2024' AND p.external_id = 'E' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'C', 3, 200, true FROM contests c, problems p
WHERE c.external_id = 'advanced-contest-2024' AND p.external_id = 'J' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'D', 4, 200, true FROM contests c, problems p
WHERE c.external_id = 'advanced-contest-2024' AND p.external_id = 'L' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'E', 5, 200, true FROM contests c, problems p
WHERE c.external_id = 'advanced-contest-2024' AND p.external_id = 'O' ON CONFLICT DO NOTHING;

-- Challenge Contest
INSERT INTO contests (external_id, name, short_name, start_time, end_time, public) VALUES
('challenge-contest-2024', 'Algorithm Challenge Contest', 'CHAL2024', NOW() + INTERVAL '1 hour', NOW() + INTERVAL '3 days', true)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'A', 1, 300, true FROM contests c, problems p
WHERE c.external_id = 'challenge-contest-2024' AND p.external_id = 'F' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'B', 2, 300, true FROM contests c, problems p
WHERE c.external_id = 'challenge-contest-2024' AND p.external_id = 'G' ON CONFLICT DO NOTHING;

INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, p.id, 'C', 3, 300, true FROM contests c, problems p
WHERE c.external_id = 'challenge-contest-2024' AND p.external_id = 'N' ON CONFLICT DO NOTHING;