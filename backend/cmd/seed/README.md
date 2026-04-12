# Database Seeding

This directory contains the database seeding system for the online judge.

## Overview

The seeding system populates the database with:
- Programming languages (C++, C, Python, Java, Go, Rust, Node.js)
- Sample problems with various difficulty levels (Easy, Medium, Hard)
- Test cases with input/output content for sample tests
- Problem statements in HTML format
- Sample contests with multiple problems

## Usage

### Primary Method (Go Seed Script)

```bash
# Run seeding (requires database connection)
make seed

# Reset database and re-seed
make seed-reset

# Build seed binary for deployment
make build-seed
```

### Fallback Method (SQL)

The `migrations/seed.sql` file provides SQL-based seeding as a fallback:
```bash
# After running migrations, apply seed.sql manually
psql -U postgres -d oj -f backend/migrations/seed.sql
```

## Sample Problems

| External ID | Name | Difficulty | Points | Test Cases |
|-------------|------|------------|--------|------------|
| A | A + B | easy | 100 | 5 (2 samples) |
| B | Sum of Array | easy | 100 | 4 (1 sample) |
| C | Fibonacci Number | easy | 100 | 5 (2 samples) |
| D | Binary Search | medium | 200 | 6 (2 samples) |
| E | Two Sum | medium | 200 | 4 (1 sample) |
| F | Longest Common Subsequence | hard | 300 | 5 (1 sample) |
| G | Shortest Path | hard | 300 | 4 (2 samples) |
| H | Maximum Subarray | medium | 200 | 4 (2 samples) |
| I | Palindrome Check | easy | 100 | 4 (2 samples) |
| J | Prime Factorization | medium | 200 | 5 (2 samples) |

## Sample Contests

| Contest ID | Name | Problems | Duration |
|------------|------|----------|----------|
| intro-contest-2024 | Introduction to Algorithms Contest | A, B, C, I | 7 days |
| advanced-contest-2024 | Advanced Algorithms Contest | D, E, H, J | 5 days |
| challenge-contest-2024 | Algorithm Challenge Contest | F, G | 3 days |

## Test Case Storage

Sample test cases are stored inline in the database (`input_content`, `output_content` columns) for easy retrieval. Hidden test cases use file paths for storage in object storage (MinIO/S3/local).

## Environment Variables

- `DATABASE_URL`: PostgreSQL connection string (default: `postgres://postgres:postgres@localhost:5432/oj?sslmode=disable`)

## Extending

To add more sample problems, modify the `sampleProblems` slice in `main.go`:

```go
{
    ExternalID: "K",
    Name:       "New Problem",
    Difficulty: "medium",
    TimeLimit:  1.0,
    MemoryLimit: 262144,
    Points:     200,
    ProblemStatement: `<h1>Problem Title</h1>...`,
    TestCases: []SampleTestCase{
        {Rank: 1, IsSample: true, Input: "...", Output: "...", Description: "..."},
    },
}
```

## Idempotency

The seed script uses `ON CONFLICT DO UPDATE` to ensure:
- Re-running the seed script won't create duplicates
- Existing data is updated with the latest seed values
- Safe to run multiple times during development