-- Seed data for testing

-- Insert languages
INSERT INTO languages (id, external_id, name, time_factor, extensions, allow_submit, allow_judge) VALUES
('cpp', 'cpp', 'C++ 17', 1.0, ARRAY['.cpp', '.cc', '.cxx'], true, true),
('c', 'c', 'C11', 1.0, ARRAY['.c'], true, true),
('python3', 'python3', 'Python 3', 2.0, ARRAY['.py', '.py3'], true, true),
('java', 'java', 'Java 17', 1.5, ARRAY['.java'], true, true),
('go', 'go', 'Go 1.21', 1.2, ARRAY['.go'], true, true),
('rust', 'rust', 'Rust', 1.0, ARRAY['.rs'], true, true),
('nodejs', 'nodejs', 'Node.js 18', 2.0, ARRAY['.js', '.mjs'], true, true);

-- Insert problems
INSERT INTO problems (id, external_id, name, time_limit, memory_limit, difficulty, points, is_published, allow_submit) VALUES
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'A', 'A + B', 1.0, 262144, 'easy', 100, true, true),
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'B', 'Sum of Array', 2.0, 262144, 'easy', 100, true, true),
('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'C', 'Fibonacci Number', 1.0, 262144, 'easy', 100, true, true),
('d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'D', 'Binary Search', 1.0, 262144, 'medium', 200, true, true),
('e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'E', 'Two Sum', 1.0, 262144, 'medium', 200, true, true),
('f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'F', 'Longest Common Subsequence', 2.0, 524288, 'hard', 300, true, true),
('00eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'G', 'Shortest Path', 1.0, 262144, 'hard', 300, true, true);

-- Insert sample test cases for "A + B"
INSERT INTO test_cases (id, problem_id, rank, is_sample, input_path, output_path, description) VALUES
(uuid_generate_v4(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1, true, 'problems/A/input/sample1.txt', 'problems/A/output/sample1.txt', 'Sample 1'),
(uuid_generate_v4(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 2, true, 'problems/A/input/sample2.txt', 'problems/A/output/sample2.txt', 'Sample 2'),
(uuid_generate_v4(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 3, false, 'problems/A/input/test1.txt', 'problems/A/output/test1.txt', 'Test 1'),
(uuid_generate_v4(), 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 4, false, 'problems/A/input/test2.txt', 'problems/A/output/test2.txt', 'Test 2');

-- Insert sample test cases for "Sum of Array"
INSERT INTO test_cases (id, problem_id, rank, is_sample, input_path, output_path, description) VALUES
(uuid_generate_v4(), 'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1, true, 'problems/B/input/sample1.txt', 'problems/B/output/sample1.txt', 'Sample 1'),
(uuid_generate_v4(), 'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 2, false, 'problems/B/input/test1.txt', 'problems/B/output/test1.txt', 'Test 1');

-- Insert sample test cases for "Fibonacci Number"
INSERT INTO test_cases (id, problem_id, rank, is_sample, input_path, output_path, description) VALUES
(uuid_generate_v4(), 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1, true, 'problems/C/input/sample1.txt', 'problems/C/output/sample1.txt', 'Sample 1'),
(uuid_generate_v4(), 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 2, true, 'problems/C/input/sample2.txt', 'problems/C/output/sample2.txt', 'Sample 2'),
(uuid_generate_v4(), 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 3, false, 'problems/C/input/test1.txt', 'problems/C/output/test1.txt', 'Test 1');

-- Insert sample test cases for "Binary Search"
INSERT INTO test_cases (id, problem_id, rank, is_sample, input_path, output_path, description) VALUES
(uuid_generate_v4(), 'd0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1, true, 'problems/D/input/sample1.txt', 'problems/D/output/sample1.txt', 'Sample 1'),
(uuid_generate_v4(), 'd0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 2, false, 'problems/D/input/test1.txt', 'problems/D/output/test1.txt', 'Test 1');

-- Insert sample test cases for "Two Sum"
INSERT INTO test_cases (id, problem_id, rank, is_sample, input_path, output_path, description) VALUES
(uuid_generate_v4(), 'e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1, true, 'problems/E/input/sample1.txt', 'problems/E/output/sample1.txt', 'Sample 1'),
(uuid_generate_v4(), 'e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 2, false, 'problems/E/input/test1.txt', 'problems/E/output/test1.txt', 'Test 1');

-- Insert sample test cases for "Longest Common Subsequence"
INSERT INTO test_cases (id, problem_id, rank, is_sample, input_path, output_path, description) VALUES
(uuid_generate_v4(), 'f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1, true, 'problems/F/input/sample1.txt', 'problems/F/output/sample1.txt', 'Sample 1'),
(uuid_generate_v4(), 'f0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 2, false, 'problems/F/input/test1.txt', 'problems/F/output/test1.txt', 'Test 1');

-- Insert sample test cases for "Shortest Path"
INSERT INTO test_cases (id, problem_id, rank, is_sample, input_path, output_path, description) VALUES
(uuid_generate_v4(), '00eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 1, true, 'problems/G/input/sample1.txt', 'problems/G/output/sample1.txt', 'Sample 1'),
(uuid_generate_v4(), '00eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 2, false, 'problems/G/input/test1.txt', 'problems/G/output/test1.txt', 'Test 1');

-- Insert a test contest
INSERT INTO contests (id, external_id, name, short_name, start_time, end_time, public, created_at)
SELECT
    'c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'test-contest-2024',
    'Test Contest 2024',
    'TC2024',
    NOW() - INTERVAL '1 day',
    NOW() + INTERVAL '7 days',
    true,
    NOW()
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'contests');

-- Add problems to contest
INSERT INTO contest_problems (id, contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT
    uuid_generate_v4(),
    'c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'A',
    1,
    100,
    true
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'contest_problems');

INSERT INTO contest_problems (id, contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT
    uuid_generate_v4(),
    'c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'B',
    2,
    100,
    true
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'contest_problems');

INSERT INTO contest_problems (id, contest_id, problem_id, short_name, rank, points, allow_submit)
SELECT
    uuid_generate_v4(),
    'c1eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'd0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'C',
    3,
    200,
    true
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'contest_problems');