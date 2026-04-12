-- Seed data for testing
-- This file provides SQL-based seeding as a fallback option.
-- The primary seeding mechanism is the Go seed script: backend/cmd/seed/main.go

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

WITH problem_seed(code, name, time_limit, memory_limit, difficulty, points, tc_input, tc_output, tc_description) AS (
    VALUES
    ('A', 'A + B', 1.0, 262144, 'easy', 100, '1 2\n', '3\n', 'Simple positive numbers'),
    ('B', 'Sum of Array', 1.0, 262144, 'easy', 100, '5\n1 2 3 4 5\n', '15\n', 'Simple positive numbers'),
    ('C', 'Fibonacci Number', 1.0, 262144, 'easy', 100, '0\n', '0\n', 'Base case F(0)'),
    ('D', 'Binary Search', 1.0, 262144, 'medium', 200, '5\n1 3 5 7 9\n5\n', '2\n', 'Found in middle'),
    ('E', 'Two Sum', 1.0, 262144, 'medium', 200, '4\n2 7 11 15\n9\n', '0 1\n', 'First two elements'),
    ('F', 'Longest Common Subsequence', 2.0, 524288, 'hard', 300, 'abcde\nace\n', '3\n', 'LCS ace'),
    ('G', 'Shortest Path', 2.0, 262144, 'hard', 300, '4 4\n1 2 1\n2 3 2\n3 4 3\n1 4 10\n', '6\n', 'Path 1-2-3-4'),
    ('H', 'Maximum Subarray', 1.0, 262144, 'medium', 200, '9\n-2 1 -3 4 -1 2 1 -5 4\n', '6\n', 'Kadane'),
    ('I', 'Palindrome Check', 1.0, 262144, 'easy', 100, 'racecar\n', 'true\n', 'Palindrome'),
    ('J', 'Prime Factorization', 1.0, 262144, 'medium', 200, '12\n', '2 2\n3 1\n', 'Prime factors'),
    ('K', 'Stack Implementation', 1.0, 262144, 'easy', 100, '5\npush 10\ntop\n', '10\n', 'Basic test'),
    ('L', 'String Matching', 1.0, 262144, 'medium', 200, 'abab\nab\n', '0 2\n', 'Multiple matches'),
    ('M', 'Sorting Algorithms', 2.0, 262144, 'easy', 100, '3\n3 1 2\n', '1 2 3\n', 'Basic sort'),
    ('N', 'Tree Traversal', 2.0, 524288, 'hard', 300, '3\n2 1 3\n1 2 3\n', '2 3 1\n', 'Simple tree'),
    ('O', 'Graph DFS', 1.0, 262144, 'medium', 200, '3 2\n1 2\n2 3\n', '1 2 3\n', 'DFS traversal')
)
INSERT INTO problems (name, time_limit, memory_limit, difficulty, points, is_published, allow_submit)
SELECT ps.name, ps.time_limit, ps.memory_limit, ps.difficulty, ps.points, true, true
FROM problem_seed ps
WHERE NOT EXISTS (
    SELECT 1 FROM problems p WHERE p.name = ps.name
);

WITH problem_seed(code, name, tc_input, tc_output, tc_description) AS (
    VALUES
    ('A', 'A + B', '1 2\n', '3\n', 'Simple positive numbers'),
    ('B', 'Sum of Array', '5\n1 2 3 4 5\n', '15\n', 'Simple positive numbers'),
    ('C', 'Fibonacci Number', '0\n', '0\n', 'Base case F(0)'),
    ('D', 'Binary Search', '5\n1 3 5 7 9\n5\n', '2\n', 'Found in middle'),
    ('E', 'Two Sum', '4\n2 7 11 15\n9\n', '0 1\n', 'First two elements'),
    ('F', 'Longest Common Subsequence', 'abcde\nace\n', '3\n', 'LCS ace'),
    ('G', 'Shortest Path', '4 4\n1 2 1\n2 3 2\n3 4 3\n1 4 10\n', '6\n', 'Path 1-2-3-4'),
    ('H', 'Maximum Subarray', '9\n-2 1 -3 4 -1 2 1 -5 4\n', '6\n', 'Kadane'),
    ('I', 'Palindrome Check', 'racecar\n', 'true\n', 'Palindrome'),
    ('J', 'Prime Factorization', '12\n', '2 2\n3 1\n', 'Prime factors'),
    ('K', 'Stack Implementation', '5\npush 10\ntop\n', '10\n', 'Basic test'),
    ('L', 'String Matching', 'abab\nab\n', '0 2\n', 'Multiple matches'),
    ('M', 'Sorting Algorithms', '3\n3 1 2\n', '1 2 3\n', 'Basic sort'),
    ('N', 'Tree Traversal', '3\n2 1 3\n1 2 3\n', '2 3 1\n', 'Simple tree'),
    ('O', 'Graph DFS', '3 2\n1 2\n2 3\n', '1 2 3\n', 'DFS traversal')
),
problem_map AS (
    SELECT DISTINCT ON (ps.code) ps.code, p.id
    FROM problem_seed ps
    JOIN problems p ON p.name = ps.name
    ORDER BY ps.code, p.created_at DESC, p.id DESC
)
INSERT INTO test_cases (problem_id, rank, is_sample, input_path, output_path, input_content, output_content, description)
SELECT
    pm.id,
    1,
    true,
    format('problems/%s/input/sample1.txt', ps.code),
    format('problems/%s/output/sample1.txt', ps.code),
    ps.tc_input,
    ps.tc_output,
    ps.tc_description
FROM problem_seed ps
JOIN problem_map pm ON pm.code = ps.code
ON CONFLICT (problem_id, rank) DO UPDATE SET
    input_content = EXCLUDED.input_content,
    output_content = EXCLUDED.output_content,
    description = EXCLUDED.description;

-- Intro Contest
INSERT INTO contests (external_id, name, short_name, start_time, end_time, public) VALUES
('intro-contest-2024', 'Introduction to Algorithms Contest', 'INTRO2024', NOW() - INTERVAL '1 day', NOW() + INTERVAL '7 days', true)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

-- Advanced Contest
INSERT INTO contests (external_id, name, short_name, start_time, end_time, public) VALUES
('advanced-contest-2024', 'Advanced Algorithms Contest', 'ADV2024', NOW() - INTERVAL '2 hours', NOW() + INTERVAL '5 days', true)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

-- Challenge Contest
INSERT INTO contests (external_id, name, short_name, start_time, end_time, public) VALUES
('challenge-contest-2024', 'Algorithm Challenge Contest', 'CHAL2024', NOW() + INTERVAL '1 hour', NOW() + INTERVAL '3 days', true)
ON CONFLICT (external_id) DO UPDATE SET
    name = EXCLUDED.name,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time;

WITH problem_seed(code, name) AS (
    VALUES
    ('A', 'A + B'),
    ('B', 'Sum of Array'),
    ('C', 'Fibonacci Number'),
    ('D', 'Binary Search'),
    ('E', 'Two Sum'),
    ('F', 'Longest Common Subsequence'),
    ('G', 'Shortest Path'),
    ('H', 'Maximum Subarray'),
    ('I', 'Palindrome Check'),
    ('J', 'Prime Factorization'),
    ('K', 'Stack Implementation'),
    ('L', 'String Matching'),
    ('M', 'Sorting Algorithms'),
    ('N', 'Tree Traversal'),
    ('O', 'Graph DFS')
),
problem_map AS (
    SELECT DISTINCT ON (ps.code) ps.code, p.id
    FROM problem_seed ps
    JOIN problems p ON p.name = ps.name
    ORDER BY ps.code, p.created_at DESC, p.id DESC
),
contest_problem_seed(contest_external_id, problem_code, short_name, rank, points) AS (
    VALUES
    ('intro-contest-2024', 'A', 'A', 1, 100),
    ('intro-contest-2024', 'B', 'B', 2, 100),
    ('intro-contest-2024', 'C', 'C', 3, 100),
    ('intro-contest-2024', 'I', 'D', 4, 100),
    ('intro-contest-2024', 'K', 'E', 5, 100),
    ('intro-contest-2024', 'M', 'F', 6, 100),
    ('advanced-contest-2024', 'D', 'A', 1, 200),
    ('advanced-contest-2024', 'E', 'B', 2, 200),
    ('advanced-contest-2024', 'J', 'C', 3, 200),
    ('advanced-contest-2024', 'L', 'D', 4, 200),
    ('advanced-contest-2024', 'O', 'E', 5, 200),
    ('challenge-contest-2024', 'F', 'A', 1, 300),
    ('challenge-contest-2024', 'G', 'B', 2, 300),
    ('challenge-contest-2024', 'N', 'C', 3, 300)
)
INSERT INTO contest_problems (contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT c.id, pm.id, cps.short_name, cps.rank, cps.points, true
FROM contest_problem_seed cps
JOIN contests c ON c.external_id = cps.contest_external_id
JOIN problem_map pm ON pm.code = cps.problem_code
ON CONFLICT (contest_id, problem_id) DO UPDATE SET
    short_name = EXCLUDED.short_name,
    rank = EXCLUDED.rank,
    points = EXCLUDED.points,
    allow_submit = EXCLUDED.allow_submit;
