-- 008_testcase_content.down.sql

ALTER TABLE test_cases DROP COLUMN IF EXISTS output_content;
ALTER TABLE test_cases DROP COLUMN IF EXISTS input_content;
DROP TABLE IF EXISTS problem_statements;