# Backend Architecture

## Overview

This document outlines the backend architecture for the Online Judge platform.

## Architecture Pattern

**Unified gRPC Server** – all domain services (Problem, Submission, Contest, User, Notification, Judge) run in a single Go process and are registered on one gRPC server on port 8002. The BFF connects to all services through the same address.

## Technology Stack

| Component           | Technology      | Justification                                                             |
| ------------------- | --------------- | ------------------------------------------------------------------------- |
| Language            | Go 1.21+        | High performance, strong concurrency, excellent for gRPC                  |
| API Protocol        | gRPC            | Strongly-typed service contracts; BFF translates to REST for the frontend |
| Protobuf Management | Buf             | Modern protobuf tooling, linting, managed mode                            |
| Database            | PostgreSQL 16   | ACID compliance, relational integrity, mature ecosystem                   |
| Cache/Queue         | Redis 7 + Asynq | High-speed caching, session management, pub/sub, judge task queue         |
| Authentication      | Identra         | Out-of-the-box auth service with JWT/JWKS, OAuth, Email, Password         |

## Project Structure

```
.                                     # repository root
├── proto/                            # Protobuf definitions (all services)
│   ├── buf.yaml
│   ├── common/v1/common.proto
│   ├── user/v1/user.proto
│   ├── problem/v1/problem.proto
│   ├── submission/v1/submission.proto
│   ├── contest/v1/contest.proto
│   ├── notification/v1/notification.proto
│   ├── judge/v1/judge.proto
│   └── sandbox/v1/sandbox.proto
├── gen/                              # Generated Go code (buf generate proto)
│   └── go/
│       ├── common/v1/
│       ├── user/v1/
│       ├── problem/v1/
│       ├── submission/v1/
│       ├── contest/v1/
│       ├── notification/v1/
│       └── judge/v1/
├── buf.gen.yaml                      # Buf generation config (project root)
└── backend/
    ├── cmd/
    │   ├── server/
    │   │   └── main.go               # Unified gRPC server entry point
    │   └── seed/
    │       └── main.go               # Database seed tool
    ├── internal/
    │   ├── problem/
    │   │   ├── service/
    │   │   └── store/
    │   ├── submission/
    │   │   ├── service/
    │   │   └── store/
    │   ├── contest/
    │   │   ├── service/
    │   │   └── store/
    │   ├── notification/
    │   │   └── service/
    │   ├── user/
    │   │   ├── service/
    │   │   └── store/
    │   ├── judge/
    │   │   ├── service/
    │   │   └── store/
    │   ├── queue/                    # Asynq + legacy Redis queue
    │   └── pkg/
    │       ├── config/               # Viper-based config loader
    │       ├── middleware/
    │       └── storage/              # Object storage (S3/MinIO/local)
    ├── migrations/                   # SQL migration files
    ├── go.mod
    └── go.sum
```

## Buf Configuration

### buf.yaml (project root `proto/buf.yaml`)

```yaml
version: v2
lint:
  use:
    - DEFAULT
breaking:
  use:
    - FILE
```

### buf.gen.yaml (project root)

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
```

## Service Definitions (Protobuf)

> Proto files live at `proto/<service>/v1/<service>.proto`.  
> Generated Go code goes to `gen/go/<service>/v1/` (module `github.com/slhmy/online-judge/gen`).

### Common Types

```protobuf
// proto/common/v1/common.proto
syntax = "proto3";

package common.v1;

option go_package = "github.com/slhmy/online-judge/gen/go/common/v1;commonv1";

message Pagination {
  int32 page = 1;
  int32 page_size = 2;
}

message PaginatedResponse {
  int32 total = 1;
  int32 page = 2;
  int32 page_size = 3;
}

message UserRef {
  string id = 1;
  string username = 2;
}
```

### User Service Proto

```protobuf
// proto/user/v1/user.proto
syntax = "proto3";

package user.v1;

option go_package = "github.com/slhmy/online-judge/gen/go/user/v1;userv1";

import "common/v1/common.proto";

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc GetUserProfile(GetUserProfileRequest) returns (GetUserProfileResponse);
  rpc GetUserStats(GetUserStatsRequest) returns (GetUserStatsResponse);
  rpc GetUserSubmissions(GetUserSubmissionsRequest) returns (GetUserSubmissionsResponse);
  rpc UpdateUserProfile(UpdateUserProfileRequest) returns (UpdateUserProfileResponse);
  rpc SyncUser(SyncUserRequest) returns (SyncUserResponse);
}
```

### Problem Service Proto

```protobuf
// proto/problem/v1/problem.proto
syntax = "proto3";

package problem.v1;

option go_package = "github.com/slhmy/online-judge/gen/go/problem/v1;problemv1";

import "common/v1/common.proto";

service ProblemService {
  rpc ListProblems(ListProblemsRequest) returns (ListProblemsResponse);
  rpc GetProblem(GetProblemRequest) returns (GetProblemResponse);
  rpc CreateProblem(CreateProblemRequest) returns (CreateProblemResponse);
  rpc UpdateProblem(UpdateProblemRequest) returns (UpdateProblemResponse);
  rpc DeleteProblem(DeleteProblemRequest) returns (DeleteProblemResponse);
  rpc GetProblemStatement(GetProblemStatementRequest) returns (GetProblemStatementResponse);
  rpc SetProblemStatement(SetProblemStatementRequest) returns (SetProblemStatementResponse);
  rpc ListLanguages(ListLanguagesRequest) returns (ListLanguagesResponse);
  rpc GetTestCases(GetTestCasesRequest) returns (GetTestCasesResponse);
}
```

### Submission Service Proto

```protobuf
// proto/submission/v1/submission.proto
syntax = "proto3";

package submission.v1;

option go_package = "github.com/slhmy/online-judge/gen/go/submission/v1;submissionv1";

import "common/v1/common.proto";

service SubmissionService {
  rpc CreateSubmission(CreateSubmissionRequest) returns (CreateSubmissionResponse);
  rpc GetSubmission(GetSubmissionRequest) returns (GetSubmissionResponse);
  rpc ListSubmissions(ListSubmissionsRequest) returns (ListSubmissionsResponse);
  rpc GetJudging(GetJudgingRequest) returns (GetJudgingResponse);
  rpc GetRuns(GetRunsRequest) returns (GetRunsResponse);
  rpc UpdateJudgingResult(UpdateJudgingResultRequest) returns (UpdateJudgingResultResponse);
}

enum Verdict {
  VERDICT_UNSPECIFIED = 0;
  VERDICT_AC = 1;   // Accepted
  VERDICT_WA = 2;   // Wrong Answer
  VERDICT_TLE = 3;  // Time Limit Exceeded
  VERDICT_MLE = 4;  // Memory Limit Exceeded
  VERDICT_RE = 5;   // Runtime Error
  VERDICT_CE = 6;   // Compilation Error
  VERDICT_PE = 7;   // Presentation Error
  VERDICT_SE = 8;   // System Error
}
```

## Server Bootstrap

All domain services are wired together and registered on a single gRPC server in `backend/cmd/server/main.go`:

```go
// backend/cmd/server/main.go
package main

import (
    "log"
    "net"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/redis/go-redis/v9"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"

    pbContest      "github.com/slhmy/online-judge/gen/go/contest/v1"
    pbJudge        "github.com/slhmy/online-judge/gen/go/judge/v1"
    pbNotification "github.com/slhmy/online-judge/gen/go/notification/v1"
    pbProblem      "github.com/slhmy/online-judge/gen/go/problem/v1"
    pbSubmission   "github.com/slhmy/online-judge/gen/go/submission/v1"
    pbUser         "github.com/slhmy/online-judge/gen/go/user/v1"

    contestService      "github.com/slhmy/online-judge/backend/internal/contest/service"
    judgeService        "github.com/slhmy/online-judge/backend/internal/judge/service"
    notificationService "github.com/slhmy/online-judge/backend/internal/notification/service"
    "github.com/slhmy/online-judge/backend/internal/pkg/config"
    problemService      "github.com/slhmy/online-judge/backend/internal/problem/service"
    "github.com/slhmy/online-judge/backend/internal/queue"
    submissionService   "github.com/slhmy/online-judge/backend/internal/submission/service"
    userService         "github.com/slhmy/online-judge/backend/internal/user/service"
    // ...stores omitted for brevity
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
    asynqClient := queue.NewAsynqClient(cfg.RedisURL)

    s := grpc.NewServer()
    pbProblem.RegisterProblemServiceServer(s, problemService.NewProblemService(...))
    pbSubmission.RegisterSubmissionServiceServer(s, submissionService.NewSubmissionService(...))
    pbContest.RegisterContestServiceServer(s, contestService.NewContestService(...))
    pbUser.RegisterUserServiceServer(s, userService.NewUserService(...))
    pbNotification.RegisterNotificationServiceServer(s, notificationService.NewNotificationService(rdb))
    pbJudge.RegisterJudgeServiceServer(s, judgeService.NewJudgeService(...))
    reflection.Register(s)

    lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }
    log.Printf("Server listening on port %s (all services unified)", cfg.GRPCPort)
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

## PostgreSQL Schema

### Users Table

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'moderator', 'admin')),
    rating INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_rating ON users(rating DESC);
```

### Problems Table (DOMjudge-style)

```sql
-- Following ICPC problem package format with DOMjudge extensions
CREATE TABLE problems (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(190) UNIQUE,          -- External identifier (e.g., "A", "B")
    name VARCHAR(255) NOT NULL,               -- Display name

    -- Time limits (DOMjudge uses seconds)
    time_limit DECIMAL(10, 3) NOT NULL DEFAULT 1.0,  -- seconds

    -- Resource limits
    memory_limit INTEGER DEFAULT 524288,      -- kilobytes (512 MB default)
    output_limit INTEGER DEFAULT 4096,        -- kilobytes (4 MB default)
    process_limit INTEGER DEFAULT 50,         -- max processes

    -- Special judging (DOMjudge validators)
    special_run_id UUID,           -- Custom run script for interactive problems
    special_compare_id UUID,       -- Custom validator (output checker)
    special_compare_args TEXT,     -- Arguments for validator

    -- Problem statement
    problem_statement_path TEXT,   -- Path to PDF/HTML in object storage

    -- Metadata
    difficulty VARCHAR(20) CHECK (difficulty IN ('easy', 'medium', 'hard')),
    color VARCHAR(20),             -- CSS color for contest display
    points INTEGER DEFAULT 1,      -- Points for partial scoring

    -- Visibility
    allow_submit BOOLEAN DEFAULT true,
    allow_judge BOOLEAN DEFAULT true,
    is_published BOOLEAN DEFAULT false,

    author_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_problems_external_id ON problems(external_id);
CREATE INDEX idx_problems_published ON problems(is_published);
CREATE INDEX idx_problems_special_run ON problems(special_run_id);
CREATE INDEX idx_problems_special_compare ON problems(special_compare_id);
```

### Test Cases Table (DOMjudge-style)

```sql
CREATE TABLE test_cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    problem_id UUID REFERENCES problems(id) ON DELETE CASCADE,

    -- Ordering and visibility
    rank INTEGER NOT NULL DEFAULT 0,           -- Execution order
    is_sample BOOLEAN DEFAULT false,           -- Visible to teams

    -- File references (stored in object storage)
    -- Following DOMjudge: input.in and output.ans
    input_path TEXT NOT NULL,                  -- S3 path to input file
    output_path TEXT NOT NULL,                 -- S3 path to expected output

    -- Checksums for caching/deduplication
    md5sum_input CHAR(32),
    md5sum_output CHAR(32),

    -- Original filename (for problem package import)
    orig_input_filename VARCHAR(255),
    orig_output_filename VARCHAR(255),

    -- Metadata
    description TEXT,                          -- Testcase description

    -- For interactive problems
    is_interactive BOOLEAN DEFAULT false,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(problem_id, rank)
);

CREATE INDEX idx_test_cases_problem ON test_cases(problem_id);
CREATE INDEX idx_test_cases_sample ON test_cases(is_sample);
```

### Executables Table (DOMjudge validators/run scripts)

```sql
-- Store special validators and run scripts
CREATE TABLE executables (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(190) UNIQUE NOT NULL,  -- e.g., "cmp_float", "interactive_xxx"
    type VARCHAR(20) NOT NULL CHECK (type IN ('run', 'compare', 'compile')),
    description TEXT,

    -- Executable stored in object storage
    executable_path TEXT NOT NULL,

    -- Metadata
    md5sum CHAR(32),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_executables_type ON executables(type);
```

### Languages Table (DOMjudge-style)

```sql
CREATE TABLE languages (
    id VARCHAR(50) PRIMARY KEY,               -- e.g., "cpp", "python3", "java"
    external_id VARCHAR(50) UNIQUE,           -- External identifier
    name VARCHAR(255) NOT NULL,               -- Display name "C++ 17"

    -- Compile script reference
    compile_script_id UUID REFERENCES executables(id),

    -- Time factor for language overhead
    time_factor DECIMAL(10, 3) DEFAULT 1.0,   -- Multiply problem timelimit

    -- Extensions and filter
    extensions TEXT[] NOT NULL,               -- ['.cpp', '.cc', '.cxx']
    filter_compiler_warnings TEXT,

    -- Flags
    allow_submit BOOLEAN DEFAULT true,
    allow_judge BOOLEAN DEFAULT true,

    -- Requirements
    require_entry_point BOOLEAN DEFAULT false,
    entry_point_description TEXT,             -- e.g., "Main-class name for Java"

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Submissions Table (DOMjudge-style)

```sql
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Core references
    user_id UUID NOT NULL,
    problem_id UUID REFERENCES problems(id),
    contest_id UUID,                          -- NULL for practice submissions
    language_id VARCHAR(50) REFERENCES languages(id),

    -- Source code (stored in object storage for large files)
    source_path TEXT NOT NULL,                -- S3 path to source archive

    -- Team priority for fair scheduling (DOMjudge)
    team_pending_count INTEGER DEFAULT 0,     -- Pending submissions by this team

    -- Metadata
    submit_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    team_id UUID,                             -- For contest submissions
    orig_submit_time TIMESTAMP WITH TIME ZONE, -- Original time if rejudged

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_submissions_user ON submissions(user_id);
CREATE INDEX idx_submissions_problem ON submissions(problem_id);
CREATE INDEX idx_submissions_contest ON submissions(contest_id);
CREATE INDEX idx_submissions_time ON submissions(submit_time DESC);
CREATE INDEX idx_submissions_team ON submissions(team_id);
```

### Judgings Table (DOMjudge-style)

```sql
CREATE TABLE judgings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID REFERENCES submissions(id),

    -- Judgehost that processed this
    judgehost_id VARCHAR(100) NOT NULL,

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    max_runtime DECIMAL(10, 3),               -- seconds
    max_memory INTEGER,                       -- kilobytes

    -- Verdict (DOMjudge style)
    verdict VARCHAR(20) CHECK (verdict IN (
        'correct', 'wrong-answer', 'timelimit',
        'memory-limit', 'run-error', 'compiler-error',
        'output-limit', 'presentation'
    )),

    -- Compilation
    compile_success BOOLEAN DEFAULT false,
    compile_output_path TEXT,                 -- S3 path to compiler output
    compile_metadata JSONB,

    -- Validity (for rejudging)
    valid BOOLEAN DEFAULT true,               -- False if superseded by rejudge

    -- Verification workflow
    verified BOOLEAN DEFAULT false,
    verified_by VARCHAR(100),
    verify_comment TEXT,
    seen BOOLEAN DEFAULT false,               -- Team has seen result

    -- For lazy judging / partial scoring
    judge_completely BOOLEAN DEFAULT false,   -- Force all testcases
    score DECIMAL(10, 3),                     -- For partial scoring

    -- UUID for compilation caching
    uuid UUID DEFAULT uuid_generate_v4(),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_judgings_submission ON judgings(submission_id);
CREATE INDEX idx_judgings_valid ON judgings(valid);
CREATE INDEX idx_judgings_verdict ON judgings(verdict);
CREATE INDEX idx_judgings_judgehost ON judgings(judgehost_id);
```

### Judging Runs Table (DOMjudge-style - per testcase results)

```sql
CREATE TABLE judging_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    judging_id UUID REFERENCES judgings(id) ON DELETE CASCADE,
    test_case_id UUID REFERENCES test_cases(id),

    -- Execution order
    rank INTEGER NOT NULL,

    -- Timing
    runtime DECIMAL(10, 3),                   -- CPU time in seconds
    wall_time DECIMAL(10, 3),                 -- Wall clock time
    memory INTEGER,                           -- kilobytes

    -- Result
    verdict VARCHAR(20) NOT NULL,

    -- Output (stored in object storage for large outputs)
    output_run_path TEXT,                     -- Program stdout
    output_diff_path TEXT,                    -- Diff with expected
    output_error_path TEXT,                   -- Program stderr
    output_system TEXT,                       -- System messages

    -- Metadata
    metadata JSONB,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_judging_runs_judging ON judging_runs(judging_id);
CREATE INDEX idx_judging_runs_testcase ON judging_runs(test_case_id);
```

### Rejudgings Table (DOMjudge-style)

```sql
CREATE TABLE rejudgings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Reason and status
    reason TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'cancelled')),

    -- Scope
    problem_id UUID REFERENCES problems(id),
    language_id VARCHAR(50) REFERENCES languages(id),
    user_id UUID,
    contest_id UUID REFERENCES contests(id),

    -- Progress
    total_submissions INTEGER DEFAULT 0,
    judged_count INTEGER DEFAULT 0,

    -- Result summary
    applied_count INTEGER DEFAULT 0,
    cancelled_count INTEGER DEFAULT 0,

    -- Auto-apply
    auto_apply BOOLEAN DEFAULT false,
    auto_apply_at TIMESTAMP WITH TIME ZONE,

    -- Owner
    created_by VARCHAR(100) NOT NULL,

    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_rejudgings_status ON rejudgings(status);
CREATE INDEX idx_rejudgings_problem ON rejudgings(problem_id);
```

### Judgehosts Table (DOMjudge-style)

```sql
CREATE TABLE judgehosts (
    id VARCHAR(100) PRIMARY KEY,              -- hostname

    -- Status
    active BOOLEAN DEFAULT true,
    status VARCHAR(20) DEFAULT 'idle' CHECK (status IN ('idle', 'judging', 'error', 'disabled')),

    -- Health
    last_ping TIMESTAMP WITH TIME ZONE,
    last_error TEXT,

    -- Configuration
    polltime INTEGER DEFAULT 10,              -- Seconds between polls

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Contests Table (DOMjudge-style)

```sql
CREATE TABLE contests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(190) UNIQUE,          -- External identifier

    -- Basic info
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),                   -- e.g., "ICPC2024"

    -- Timing
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    freeze_time TIMESTAMP WITH TIME ZONE,     -- Scoreboard freeze
    unfreeze_time TIMESTAMP WITH TIME ZONE,   -- Scoreboard unfreeze
    finalize_time TIMESTAMP WITH TIME ZONE,   -- Contest finalized

    -- Settings
    activation_time TIMESTAMP WITH TIME ZONE,
    public BOOLEAN DEFAULT true,              -- Visible publicly

    -- Contest file (contest.yaml)
    contest_file_path TEXT,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_contests_time ON contests(start_time, end_time);
CREATE INDEX idx_contests_public ON contests(public);
```

### Contest Problems Table

```sql
CREATE TABLE contest_problems (
    contest_id UUID REFERENCES contests(id) ON DELETE CASCADE,
    problem_id UUID REFERENCES problems(id),

    -- Order and display
    short_name VARCHAR(10) NOT NULL,          -- e.g., "A", "B", "C"
    rank INTEGER NOT NULL,                    -- Display order
    color VARCHAR(20),                        -- CSS color

    -- Points for this problem in this contest
    points INTEGER DEFAULT 1,

    -- Visibility
    allow_submit BOOLEAN DEFAULT true,
    allow_judge BOOLEAN DEFAULT true,

    PRIMARY KEY (contest_id, problem_id)
);

CREATE INDEX idx_contest_problems_contest ON contest_problems(contest_id);
```

### Scoreboard Cache (Redis)

```sql
-- Scoreboard data is cached in Redis, but we store team scores
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE REFERENCES user_profiles(user_id),

    -- Team info
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),

    -- Affiliation
    affiliation VARCHAR(255),
    country VARCHAR(3),                       -- ISO 3166-1 alpha-3

    -- Contest specific
    contest_id UUID REFERENCES contests(id),

    -- Scoring
    points INTEGER DEFAULT 0,
    total_time INTEGER DEFAULT 0,             -- Penalty time in minutes

    -- Location
    room VARCHAR(255),
    location VARCHAR(255),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_teams_contest ON teams(contest_id);
CREATE INDEX idx_teams_user ON teams(user_id);
```

    registered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (contest_id, user_id)

);

````

## Redis Usage

### Session Management
```go
// Store JWT token with user info
func (c *Cache) SetSession(token string, userID string, expiry time.Duration) error {
    key := fmt.Sprintf("session:%s", token)
    return c.client.Set(ctx, key, userID, expiry).Err()
}

func (c *Cache) GetSession(token string) (string, error) {
    key := fmt.Sprintf("session:%s", token)
    return c.client.Get(ctx, key).Result()
}
````

### Problem Cache

```go
// Cache problem data for fast retrieval
func (c *Cache) SetProblem(problemID string, problem *Problem) error {
    key := fmt.Sprintf("problem:%s", problemID)
    data, _ := json.Marshal(problem)
    return c.client.Set(ctx, key, data, 10*time.Minute).Err()
}

func (c *Cache) GetProblem(problemID string) (*Problem, error) {
    key := fmt.Sprintf("problem:%s", problemID)
    data, err := c.client.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    var problem Problem
    json.Unmarshal([]byte(data), &problem)
    return &problem, nil
}
```

### Leaderboard Cache

```go
// Contest leaderboard using Redis sorted sets
func (c *Cache) UpdateLeaderboard(contestID string, userID string, score float64) error {
    key := fmt.Sprintf("leaderboard:%s", contestID)
    return c.client.ZAdd(ctx, key, &redis.Z{Score: score, Member: userID}).Err()
}

func (c *Cache) GetLeaderboard(contestID string, limit int64) ([]LeaderboardEntry, error) {
    key := fmt.Sprintf("leaderboard:%s", contestID)
    result, err := c.client.ZRevRangeWithScores(ctx, key, 0, limit-1).Result()
    // ...
}
```

## Service Ports

| Service              | gRPC Port | HTTP Port | Description                          |
| -------------------- | --------- | --------- | ------------------------------------ |
| Identra Auth         | 50051     | 8081      | Authentication, JWT, OAuth, Password |
| Problem Service      | 8002      | -         | Problem CRUD, test cases             |
| Submission Service   | 8003      | -         | Submissions, results                 |
| Contest Service      | 8004      | -         | Contests, rankings                   |
| Notification Service | 8005      | -         | WebSocket, notifications             |
| gRPC-Gateway         | -         | 8080      | HTTP/REST API endpoint               |

## Makefile Commands

```makefile
.PHONY: all proto build run clean

# Generate protobuf with Buf
proto:
	buf generate

# Build all services
build:
	go build -o bin/user-service ./cmd/user-service
	go build -o bin/problem-service ./cmd/problem-service
	go build -o bin/submission-service ./cmd/submission-service
	go build -o bin/contest-service ./cmd/contest-service
	go build -o bin/notification-service ./cmd/notification-service
	go build -o bin/gateway ./cmd/gateway

# Run with Docker Compose
run:
	docker-compose up -d

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf gen/

# Database migrations
migrate-up:
	migrate -path migrations -database "postgres://..." up

migrate-down:
	migrate -path migrations -database "postgres://..." down

# Lint protobuf
lint:
	buf lint

# Format protobuf
format:
	buf format -w
```

## Authentication with Identra

We use [Identra](https://github.com/poly-workshop/identra) as a standalone authentication service that handles:

- **OAuth (GitHub)** - Social login
- **Email Code** - Passwordless email authentication
- **Password** - Traditional email/password login
- **JWT/JWKS** - Industry-standard token management

### Architecture Integration

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (Next.js)                       │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              │ 1. Login via Identra HTTP Gateway
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                    Identra Auth Service                          │
│  gRPC: 50051 | HTTP Gateway: 8081                               │
│                                                                  │
│  Features:                                                       │
│  - OAuth (GitHub)                                               │
│  - Email Code Authentication                                    │
│  - Password Authentication                                      │
│  - JWT Token Generation                                         │
│  - JWKS Endpoint for Token Validation                           │
│  - Token Refresh                                                │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              │ 2. Returns JWT Token
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                         Backend Services                         │
│                                                                  │
│  All services validate JWT using Identra JWKS:                  │
│  GET /.well-known/jwks.json                                     │
└─────────────────────────────────────────────────────────────────┘
```

### Identra Configuration

```toml
# configs/grpc/default.toml
grpc_port = 50051

[auth]
jwt_secret = "your-secret-key-change-in-production"
oauth_state_expiration = "10m"
token_expiration = "24h"

[redis]
urls = "redis:6379"

[persistence.gorm]
driver = "postgres"
host = "postgres"
port = 5432
name = "identra"
user = "identra_user"
password = "secure_password"
sslmode = "disable"

[smtp_mailer]
host = "smtp.example.com"
port = 587
username = "noreply@online-judge.com"
password = "smtp-password"
from = "noreply@online-judge.com"

[oauth.github]
client_id = "your-github-client-id"
client_secret = "your-github-client-secret"
redirect_url = "http://localhost:3000/oauth/callback"
```

### JWT Token Validation in Go Services

All backend services validate JWT tokens using Identra's JWKS endpoint:

```go
// internal/pkg/auth/jwt_validator.go
package auth

import (
    "context"
    "fmt"

    "github.com/golang-jwt/jwt/v5"
    "github.com/lestrrat-go/jwx/jwk"
)

type JWTValidator struct {
    jwksURL string
    keySet  jwk.Set
}

func NewJWTValidator(jwksURL string) (*JWTValidator, error) {
    // Fetch and cache JWKS
    set, err := jwk.Fetch(context.Background(), jwksURL)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
    }

    return &JWTValidator{
        jwksURL: jwksURL,
        keySet:  set,
    }, nil
}

// ValidateToken validates JWT and returns user claims
func (v *JWTValidator) ValidateToken(tokenString string) (*UserClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        // Verify signing algorithm
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }

        // Get key ID from token header
        kid, ok := token.Header["kid"].(string)
        if !ok {
            return nil, fmt.Errorf("kid header not found")
        }

        // Find matching key in JWKS
        key, ok := v.keySet.LookupKeyID(kid)
        if !ok {
            return nil, fmt.Errorf("key with kid %s not found", kid)
        }

        // Extract public key
        var pubkey interface{}
        if err := key.Raw(&pubkey); err != nil {
            return nil, fmt.Errorf("failed to extract public key: %w", err)
        }

        return pubkey, nil
    })

    if err != nil {
        return nil, err
    }

    if !token.Valid {
        return nil, fmt.Errorf("token is invalid")
    }

    // Extract claims
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, fmt.Errorf("failed to extract claims")
    }

    return &UserClaims{
        UserID:   claims["user_id"].(string),
        Email:    claims["email"].(string),
        Issuer:   claims["iss"].(string),
        IssuedAt: claims["iat"].(int64),
        ExpiresAt: claims["exp"].(int64),
    }, nil
}

type UserClaims struct {
    UserID    string
    Email     string
    Issuer    string
    IssuedAt  int64
    ExpiresAt int64
}
```

### gRPC Auth Interceptor

```go
// internal/common/middleware/auth_interceptor.go
package middleware

import (
    "context"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"

    "github.com/slhmy/online-judge/backend/internal/pkg/auth"
)

type AuthInterceptor struct {
    validator *auth.JWTValidator
}

func NewAuthInterceptor(validator *auth.JWTValidator) *AuthInterceptor {
    return &AuthInterceptor{validator: validator}
}

// Unary interceptor for JWT validation
func (i *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // Skip auth for public endpoints
        if isPublicEndpoint(info.FullMethod) {
            return handler(ctx, req)
        }

        // Extract token from metadata
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "metadata not found")
        }

        authHeader := md.Get("authorization")
        if len(authHeader) == 0 {
            return nil, status.Error(codes.Unauthenticated, "authorization header not found")
        }

        // Parse Bearer token
        token := extractBearerToken(authHeader[0])
        if token == "" {
            return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
        }

        // Validate token
        claims, err := i.validator.ValidateToken(token)
        if err != nil {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }

        // Add user info to context
        ctx = context.WithValue(ctx, "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "user_email", claims.Email)

        return handler(ctx, req)
    }
}

func isPublicEndpoint(method string) bool {
    publicMethods := []string{
        "/problem.v1.ProblemService/ListProblems",
        "/problem.v1.ProblemService/GetProblem",
        "/contest.v1.ContestService/ListContests",
    }
    for _, m := range publicMethods {
        if method == m {
            return true
        }
    }
    return false
}

func extractBearerToken(header string) string {
    if len(header) < 7 || header[:7] != "Bearer " {
        return ""
    }
    return header[7:]
}
```

### Service Bootstrap with Auth

```go
// cmd/problem-service/main.go
package main

import (
    "log"
    "net"

    "github.com/slhmy/online-judge/backend/internal/common/config"
    "github.com/slhmy/online-judge/backend/internal/common/middleware"
    "github.com/slhmy/online-judge/backend/internal/pkg/auth"
    "github.com/slhmy/online-judge/backend/internal/pkg/db"
    "github.com/slhmy/online-judge/backend/internal/problem/service"
    "github.com/slhmy/online-judge/backend/internal/problem/store"
    problemv1 "github.com/slhmy/online-judge/backend/gen/go/problem/v1"

    "google.golang.org/grpc"
)

func main() {
    cfg := config.Load()

    // PostgreSQL connection
    pgDB := db.NewPostgres(cfg.DatabaseURL)
    problemStore := store.NewProblemStore(pgDB)

    // JWT validator (connects to Identra JWKS)
    validator, err := auth.NewJWTValidator(cfg.IdentraJWKSURL)
    if err != nil {
        log.Fatalf("failed to create JWT validator: %v", err)
    }

    // Auth interceptor
    authInterceptor := middleware.NewAuthInterceptor(validator)

    // gRPC server with auth middleware
    srv := grpc.NewServer(
        grpc.UnaryInterceptor(authInterceptor.Unary()),
    )

    problemService := service.NewProblemService(problemStore)
    problemv1.RegisterProblemServiceServer(srv, problemService)

    lis, err := net.Listen("tcp", ":" + cfg.GRPCPort)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    log.Printf("Problem Service listening on %s", cfg.GRPCPort)
    if err := srv.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}
```

### Identra Client in BFF

```go
// bff/internal/grpc/identra_client.go
package grpc

import (
    "context"

    identrav1 "github.com/poly-workshop/identra/gen/go/identra/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

type IdentraClient struct {
    client identrav1.IdentraServiceClient
    conn   *grpc.ClientConn
}

func NewIdentraClient(addr string) (*IdentraClient, error) {
    conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, err
    }

    return &IdentraClient{
        client: identrav1.NewIdentraServiceClient(conn),
        conn:   conn,
    }, nil
}

// OAuth login via Identra
func (c *IdentraClient) OAuthLogin(ctx context.Context, code, state string) (*TokenResponse, error) {
    resp, err := c.client.OAuthLogin(ctx, &identrav1.OAuthLoginRequest{
        Provider: "github",
        Code:     code,
        State:    state,
    })
    if err != nil {
        return nil, err
    }

    return &TokenResponse{
        AccessToken:  resp.Token.AccessToken.Token,
        RefreshToken: resp.Token.RefreshToken.Token,
        ExpiresIn:    resp.Token.AccessToken.ExpiresIn,
    }, nil
}

// Password login via Identra
func (c *IdentraClient) PasswordLogin(ctx context.Context, email, password string) (*TokenResponse, error) {
    resp, err := c.client.PasswordLogin(ctx, &identrav1.PasswordLoginRequest{
        Email:    email,
        Password: password,
    })
    if err != nil {
        return nil, err
    }

    return &TokenResponse{
        AccessToken:  resp.Token.AccessToken.Token,
        RefreshToken: resp.Token.RefreshToken.Token,
        ExpiresIn:    resp.Token.AccessToken.ExpiresIn,
    }, nil
}

// Refresh token
func (c *IdentraClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
    resp, err := c.client.RefreshToken(ctx, &identrav1.RefreshTokenRequest{
        RefreshToken: refreshToken,
    })
    if err != nil {
        return nil, err
    }

    return &TokenResponse{
        AccessToken:  resp.Token.AccessToken.Token,
        ExpiresIn:    resp.Token.AccessToken.ExpiresIn,
    }, nil
}

// Get user info
func (c *IdentraClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
    resp, err := c.client.GetMeLoginInfo(ctx, &identrav1.GetMeLoginInfoRequest{
        AccessToken: accessToken,
    })
    if err != nil {
        return nil, err
    }

    return &UserInfo{
        UserID:  resp.UserId,
        Email:   resp.Email,
    }, nil
}

func (c *IdentraClient) Close() error {
    return c.conn.Close()
}

type TokenResponse struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int64
}

type UserInfo struct {
    UserID string
    Email  string
}
```

### BFF Auth Handler

```go
// bff/internal/handler/auth.go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/slhmy/online-judge/bff/internal/grpc"
)

type AuthHandler struct {
    identraClient *grpc.IdentraClient
}

// OAuth URL - redirect user to GitHub
func (h *AuthHandler) GetOAuthURL(c *gin.Context) {
    provider := c.Query("provider")
    if provider == "" {
        provider = "github"
    }

    // Call Identra to get OAuth URL
    url, state := h.identraClient.GetOAuthURL(provider)

    c.JSON(http.StatusOK, gin.H{
        "url":   url,
        "state": state,
    })
}

// OAuth callback - exchange code for token
func (h *AuthHandler) OAuthCallback(c *gin.Context) {
    code := c.Query("code")
    state := c.Query("state")

    tokens, err := h.identraClient.OAuthLogin(c.Request.Context(), code, state)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "oauth login failed"})
        return
    }

    // Set tokens in response
    c.JSON(http.StatusOK, gin.H{
        "access_token":  tokens.AccessToken,
        "refresh_token": tokens.RefreshToken,
        "expires_in":    tokens.ExpiresIn,
    })
}

// Password login
func (h *AuthHandler) Login(c *gin.Context) {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }

    tokens, err := h.identraClient.PasswordLogin(c.Request.Context(), req.Email, req.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "login failed"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "access_token":  tokens.AccessToken,
        "refresh_token": tokens.RefreshToken,
        "expires_in":    tokens.ExpiresIn,
    })
}

// Refresh token
func (h *AuthHandler) Refresh(c *gin.Context) {
    var req struct {
        RefreshToken string `json:"refresh_token"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }

    tokens, err := h.identraClient.RefreshToken(c.Request.Context(), req.RefreshToken)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh failed"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "access_token": tokens.AccessToken,
        "expires_in":   tokens.ExpiresIn,
    })
}

// Get current user info
func (h *AuthHandler) GetMe(c *gin.Context) {
    token := c.GetHeader("Authorization")
    if token == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "no token"})
        return
    }

    // Remove "Bearer " prefix
    token = extractBearerToken(token)

    userInfo, err := h.identraClient.GetUserInfo(c.Request.Context(), token)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
        return
    }

    c.JSON(http.StatusOK, userInfo)
}
```

### Auth Endpoints (BFF)

```
GET    /api/v1/auth/oauth/url        → Redirect to OAuth provider
GET    /api/v1/auth/oauth/callback   → Handle OAuth callback
POST   /api/v1/auth/login            → Password login
POST   /api/v1/auth/register         → Password registration (via Identra)
POST   /api/v1/auth/refresh          → Refresh access token
GET    /api/v1/auth/me               → Get current user info
GET    /api/v1/auth/jwks             → JWKS for token validation (proxied from Identra)
```

### User Profile Service (Optional)

If additional user profile data (rating, solved problems) is needed beyond identra's auth data:

```sql
-- User profiles table (separate from identra's user table)
CREATE TABLE user_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,  -- References identra user_id
    username VARCHAR(50) UNIQUE NOT NULL,
    rating INTEGER DEFAULT 0,
    solved_count INTEGER DEFAULT 0,
    submission_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_user_profiles_user ON user_profiles(user_id);
CREATE INDEX idx_user_profiles_rating ON user_profiles(rating DESC);
```

## Inter-Service Communication

### gRPC (Internal)

```go
// Call Problem Service from Submission Service
func (s *SubmissionService) GetProblem(ctx context.Context, problemID string) (*Problem, error) {
    conn, err := grpc.Dial("problem-service:8002", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, err
    }
    defer conn.Close()

    client := problemv1.NewProblemServiceClient(conn)
    resp, err := client.GetProblem(ctx, &problemv1.GetProblemRequest{Id: problemID})
    // ...
}
```

### Redis Queue (Async)

```go
// Submission queue for judge system (Redis sorted set)
func (q *Queue) PushSubmission(ctx context.Context, submission *Submission) error {
    data, _ := json.Marshal(submission)

    // Use sorted set for priority queue
    // Lower score = higher priority
    score := float64(submission.Priority*1000000) - float64(submission.SubmitTime.Unix())

    return q.redis.ZAdd(ctx, "judge:queue", &redis.Z{
        Score:  score,
        Member: data,
    }).Err()
}

// Pop submission from queue (called by judge workers)
func (q *Queue) PopSubmission(ctx context.Context) (*Submission, error) {
    result, err := q.redis.ZPopMin(ctx, "judge:queue").Result()
    if err != nil || len(result) == 0 {
        return nil, err
    }

    var submission Submission
    json.Unmarshal([]byte(result[0].Member.(string)), &submission)
    return &submission, nil
}

// Subscribe to judge results (Redis pub/sub)
func (q *Queue) SubscribeResults(ctx context.Context) <-chan *JudgeResult {
    ch := make(chan *JudgeResult, 100)
    pubsub := q.redis.Subscribe(ctx, "judge:results")

    go func() {
        defer close(ch)
        for msg := range pubsub.Channel() {
            var result JudgeResult
            json.Unmarshal([]byte(msg.Payload), &result)
            ch <- &result
        }
    }()

    return ch
}
```

## Monitoring & Observability

- **Metrics**: Prometheus + Grafana (gRPC metrics via grpc-prometheus)
- **Logging**: Zap (structured logging)
- **Tracing**: OpenTelemetry (distributed tracing)
- **Health**: gRPC health checking protocol

## Next Steps

1. Initialize Go module and Buf workspace
2. Define protobuf schemas for all services
3. Set up PostgreSQL with migrations
4. Implement User Service (auth + profile)
5. Implement Problem Service
6. Implement Submission Service + Redis queue integration
7. Set up gRPC-Gateway
8. Connect to Judging System
