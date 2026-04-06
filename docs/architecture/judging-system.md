# Judging System Architecture

> Based on [DOMjudge](https://github.com/DOMjudge/domjudge) - the proven ICPC contest system

## Overview

This document outlines the architecture for the Online Judge execution system, following DOMjudge's battle-tested design patterns used in ICPC regional and world finals contests.

## Core Principles (from DOMjudge)

1. **Security**: Sandboxed execution via cgroups and chroot
2. **Consistency**: Reproducible judging with controlled environment
3. **Scalability**: Distributed judgehosts with parallel processing
4. **Fairness**: Priority queue prevents team starvation
5. **Flexibility**: Support for custom validators and special judging

## Technology Stack

| Component | Technology | Justification |
|-----------|------------|---------------|
| Runtime | Go | High performance, excellent concurrency |
| Container Runtime | Docker + cgroups | Resource control, process isolation |
| Sandbox | runguard-style wrapper | CPU/memory/time enforcement |
| Database | PostgreSQL | Same as backend - relational data integrity |
| Queue/Cache/PubSub | Redis | Judge queue, caching, real-time updates |
| Object Storage | MinIO / S3 / Local | Test case files, submission archives |
| Monitoring | Prometheus | Metrics collection, alerting |

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      Judge Orchestrator                          │
│  (domserver equivalent)                                          │
│  - Submission queue management (Redis ZSET)                      │
│  - Worker coordination                                           │
│  - Result aggregation                                            │
│  - Verification workflow                                         │
└─────────────────────────────┬───────────────────────────────────┘
                              │
                              │ Redis
                              │ (Sorted Set for priority queue)
                              │
         ┌────────────────────┼────────────────────┐
         │                    │                    │
    ┌────▼────┐          ┌────▼────┐         ┌────▼────┐
    │Judgehost│          │Judgehost│         │Judgehost│
    │    1    │          │    2    │         │    N    │
    │         │          │         │         │         │
    │ ┌─────┐ │          │ ┌─────┐ │         │ ┌─────┐ │
    │ │judge│ │          │ │judge│ │         │ │judge│ │
    │ │daemon│          │ │daemon│ │         │ │daemon│ │
    │ └──┬──┘ │          │ └──┬──┘ │         │ └──┬──┘ │
    └───┼────┘          └───┼────┘         └───┼────┘
        │                   │                  │
    ┌───▼───────────────────▼──────────────────▼───┐
    │              Docker + cgroups                 │
    │  - Memory limits                              │
    │  - CPU time limits                            │
    │  - Process limits                             │
    │  - Network isolation                          │
    └───────────────────────────────────────────────┘
```

## Submission Flow (DOMjudge-style)

```
1. SUBMISSION RECEIVED
   └──► Basic validation (file size, language, problem)
   └──► Check team's pending submissions (anti-spam)
   └──► Store submission with status "pending"
   └──► Push to judge queue

2. JUDGEHOST PICKS UP
   └──► Request work from orchestrator
   └──► Download submission + testcases
   └──► Create judging record

3. COMPILATION
   └──► Create isolated environment
   └──► Run compile script
   └──► Check exit code
       ├──► Failure: Return "compiler-error", store output
       └──► Success: Cache executable, continue

4. EXECUTION (per testcase)
   └──► For each testcase (ordered by rank):
        ├──► Set up sandbox (cgroups, chroot)
        ├──► Set resource limits (CPU, memory, output)
        ├──► Provide input via stdin
        ├──► Execute with runguard wrapper
        ├──► Capture stdout/stderr
        ├──► Check verdict:
        │    ├──► time-limit: CPU or wall time exceeded
        │    ├──► memory-limit: RSS exceeded limit
        │    ├──► run-error: Non-zero exit / signal
        │    └──► OK: Continue to validation
        ├──► Run validator (default or special):
        │    ├──► correct: Output accepted
        │    ├──► wrong-answer: Output rejected
        │    └──► presentation-error: Format issue
        └──► LAZY JUDGING: Stop on highest priority verdict
             (errors stop early, correct continues)

5. RESULT AGGREGATION
   └──► Calculate max runtime and memory
   └──► Determine final verdict (highest priority)
   └──► Store judging result
   └──► Update submission status
   └──► Send WebSocket notification

6. VERIFICATION (optional)
   └──► If verification required:
        │    └──► Wait for jury member to verify
        │    └──► Only then show to team
   └──► Else: Auto-verify and show result
```

## Verdict Types (DOMjudge-compatible)

```go
type Verdict string

const (
    // Standard verdicts
    VerdictCorrect       Verdict = "correct"        // Accepted
    VerdictWrongAnswer   Verdict = "wrong-answer"   // Wrong Answer
    VerdictTimeLimit     Verdict = "timelimit"      // Time Limit Exceeded
    VerdictMemoryLimit   Verdict = "memory-limit"   // Memory Limit Exceeded
    VerdictRunError      Verdict = "run-error"      // Runtime Error
    VerdictCompilerError Verdict = "compiler-error" // Compilation Error
    
    // Additional verdicts
    VerdictOutputLimit   Verdict = "output-limit"   // Output Limit Exceeded
    VerdictPresentation  Verdict = "presentation"   // Presentation Error
    
    // System verdicts
    VerdictInternalError Verdict = "internal-error" // Judge Error
)

// Verdict priority for lazy judging (higher = stop early)
var VerdictPriority = map[Verdict]int{
    VerdictCorrect:       1,
    VerdictPresentation:  2,
    VerdictWrongAnswer:   3,
    VerdictOutputLimit:   4,
    VerdictRunError:      5,
    VerdictMemoryLimit:   6,
    VerdictTimeLimit:     7,
    VerdictCompilerError: 8,
}
```

## Time Limit Enforcement (DOMjudge-style)

DOMjudge uses both soft and hard time limits:

```go
type TimeLimits struct {
    SoftCPUTime   float64 // The problem's time limit
    HardCPUTime   float64 // Soft + overshoot
    SoftWallTime  float64 // Usually same as soft CPU
    HardWallTime  float64 // Soft + overshoot
}

// Calculate time limits with overshoot
func CalculateTimeLimits(baseTimeLimit float64, cfg *Config) TimeLimits {
    // Overshoot can be absolute + relative
    // Example: 2 seconds + 10% of base limit
    hardCPUTime := baseTimeLimit + cfg.TimeLimitOvershoot.Absolute
    hardCPUTime += baseTimeLimit * cfg.TimeLimitOvershoot.Relative
    
    return TimeLimits{
        SoftCPUTime:  baseTimeLimit,
        HardCPUTime:  hardCPUTime,
        SoftWallTime: baseTimeLimit * 1.5, // Wall time more lax
        HardWallTime: hardCPUTime * 1.5,
    }
}

// runguard-style enforcement
func (r *Runguard) Execute(cmd *exec.Cmd, limits TimeLimits) (*ExecutionResult, error) {
    // Set cgroup limits
    cgroup := cgroups.NewCgroup()
    cgroup.SetCPULimit(limits.HardCPUTime)
    cgroup.SetMemoryLimit(r.MemoryLimit)
    
    // Run with timeout monitoring
    startTime := time.Now()
    err := cmd.Run()
    elapsed := time.Since(startTime)
    
    result := &ExecutionResult{
        WallTime: elapsed.Seconds(),
        CPUTime:  cgroup.GetCPUUsage(),
        Memory:   cgroup.GetMemoryUsage(),
    }
    
    // Determine if soft/hard limit was reached
    if result.CPUTime > limits.HardCPUTime {
        result.TimelimitReached = "hard"
    } else if result.CPUTime > limits.SoftCPUTime {
        result.TimelimitReached = "soft"
    }
    
    return result, nil
}
```

## Resource Limits (DOMjudge Configuration)

```yaml
# Default limits (can be overridden per problem)
limits:
  timelimit_overshoot:
    absolute: 2.0      # seconds
    relative: 0.1      # 10%
  
  memory_limit: 524288  # 512 MB default
  output_limit: 4096    # 4 MB output limit
  process_limit: 50     # Max processes/threads
  
  disk_space_error: 1073741824  # 1GB minimum free
```

## Testcase Design (DOMjudge-style)

### Testcase Structure

```go
type Testcase struct {
    ID          string    `json:"id"`
    ProblemID   string    `json:"problem_id"`
    Rank        int       `json:"rank"`         // Order of execution
    IsSample    bool      `json:"is_sample"`    // Visible to teams
    
    // Input/Output stored in object storage
    InputFile   string    `json:"input_file"`   // S3 path to input
    OutputFile  string    `json:"output_file"`  // S3 path to expected output
    
    // Metadata
    MD5Input    string    `json:"md5_input"`    // Checksum for caching
    MD5Output   string    `json:"md5_output"`
    Description string    `json:"description"`  // Optional description
    
    // For interactive problems
    IsInteractive bool    `json:"is_interactive"`
}
```

### Testcase Execution Order

```
Testcases run in order by rank:
  Rank 1 (sample) → Rank 2 (sample) → Rank 3 (secret) → ...
  
Sample testcases:
  - Visible to teams before submission
  - Can be downloaded for practice
  - Usually simpler, verify basic understanding

Secret testcases:
  - Not visible to teams
  - Test edge cases, performance, correctness
  - Multiple secret testcases per problem
```

## Output Validation

### Default Validator (DOMjudge/Kattis style)

```go
// Default output comparison
func DefaultValidator(expected, actual string, config ValidatorConfig) Verdict {
    // Normalize line endings
    expected = strings.ReplaceAll(expected, "\r\n", "\n")
    actual = strings.ReplaceAll(actual, "\r\n", "\n")
    
    // Apply normalization based on config
    if config.IgnoreTrailingWhitespace {
        expected = normalizeTrailingWhitespace(expected)
        actual = normalizeTrailingWhitespace(actual)
    }
    
    if config.IgnoreCase {
        expected = strings.ToLower(expected)
        actual = strings.ToLower(actual)
    }
    
    // Exact match
    if expected == actual {
        return VerdictCorrect
    }
    
    // Check presentation error (whitespace differences)
    if isPresentationError(expected, actual, config) {
        return VerdictPresentation
    }
    
    return VerdictWrongAnswer
}

func normalizeTrailingWhitespace(s string) string {
    lines := strings.Split(s, "\n")
    for i, line := range lines {
        lines[i] = strings.TrimRight(line, " \t")
    }
    return strings.Join(lines, "\n")
}
```

### Special Validator (Custom Compare Script)

```go
type SpecialValidator struct {
    ID          string
    Executable  []byte    // Compiled validator binary
    Language    string
    Args        []string  // Additional arguments
}

// Run custom validator
func (v *SpecialValidator) Validate(input, expected, actual string) (Verdict, string) {
    // Create temp files
    inputFile := writeFile(input)
    expectedFile := writeFile(expected)
    actualFile := writeFile(actual)
    
    // Run validator
    // Exit codes: 42 = correct, 43 = wrong-answer, other = internal-error
    cmd := exec.Command(v.ExecutablePath, 
        inputFile, expectedFile, actualFile,
        v.Args...)
    
    output, err := cmd.CombinedOutput()
    
    switch cmd.ProcessState.ExitCode() {
    case 42:
        return VerdictCorrect, string(output)
    case 43:
        return VerdictWrongAnswer, string(output)
    default:
        return VerdictInternalError, string(output)
    }
}
```

### Special Run Script (Interactive Problems)

```go
type SpecialRun struct {
    ID         string
    Executable []byte
    Language   string
}

// Run interactive problem
func (r *SpecialRun) Execute(solutionBinary string, testcase *Testcase) (*ExecutionResult, Verdict) {
    // Run both the solution and the interactive runner
    // The runner communicates with the solution via pipes
}
```

## Priority Queue (DOMjudge Team Fairness)

```go
// DOMjudge prevents team starvation by prioritizing
// submissions from teams with fewer pending submissions

type SubmissionPriority struct {
    TeamPendingCount int    // Number of pending submissions for this team
    SubmitTime       time.Time
    IsContest        bool   // Contest submissions higher priority
}

// Calculate priority (lower = higher priority)
func (p SubmissionPriority) Priority() int {
    // Teams with more pending submissions get lower priority
    // This prevents a team from flooding the queue
    basePriority := p.TeamPendingCount * 10
    
    if p.IsContest {
        basePriority -= 100  // Contest submissions prioritized
    }
    
    return basePriority
}
```

## Rejudging System (DOMjudge-style)

```go
type Rejudging struct {
    ID           string
    Reason       string
    Status       string    // "pending", "processing", "completed"
    Submissions  []string  // Submission IDs to rejudge
    CreatedBy    string
    
    // For batched rejudging
    ApplyAutomatically bool  // Auto-apply new results
    AutoApplyAt        *time.Time
}

// Rejudging flow
func (r *Rejudging) Process() {
    for _, subID := range r.Submissions {
        // Create new judging for old submission
        oldSubmission := GetSubmission(subID)
        
        // Mark old judgings as invalid
        MarkOldJudgingsInvalid(subID)
        
        // Requeue for judging
        SubmitToJudgeQueue(oldSubmission)
    }
}

// Apply rejudging results
func (r *Rejudging) Apply() {
    for _, subID := range r.Submissions {
        newJudging := GetLatestJudging(subID)
        oldJudging := GetPreviousJudging(subID)
        
        // Update scoreboard if verdict changed
        if newJudging.Verdict != oldJudging.Verdict {
            UpdateScoreboard(subID, newJudging.Verdict)
        }
    }
}
```

## Verification Workflow

```go
type Judging struct {
    ID           string
    SubmissionID string
    Verdict      Verdict
    Verified     bool      // Has jury reviewed?
    VerifiedBy   string    // Jury member who verified
    VerifyComment string   // Optional comment
    
    // For requiring verification before showing to team
    RequireVerification bool
}

// If verification_required is enabled:
func (j *Judging) IsVisibleToTeam() bool {
    return j.Verified && j.Valid
}
```

## Lazy Judging

```go
// Lazy judging stops early when highest priority verdict is found
// This saves resources since many submissions have errors

type LazyJudgingConfig struct {
    Enabled     bool
    // Verdicts with higher priority stop judging early
    // "correct" has lowest priority (must run all testcases)
}

func (j *JudgeWorker) RunTestcases(testcases []Testcase, cfg LazyJudgingConfig) Verdict {
    highestPriority := 0
    finalVerdict := VerdictCorrect
    
    for _, tc := range testcases {
        result := j.RunTestcase(tc)
        
        priority := VerdictPriority[result.Verdict]
        
        // If lazy judging enabled and we found high priority verdict
        if cfg.Enabled && priority > highestPriority {
            highestPriority = priority
            finalVerdict = result.Verdict
            
            // Stop early for error verdicts (not correct)
            if result.Verdict != VerdictCorrect {
                break
            }
        }
    }
    
    return finalVerdict
}
```

## Storage Architecture

Using the same stack as backend services: PostgreSQL + Redis + MinIO/S3/Local

### PostgreSQL (Metadata)
- Submission records
- Judging results
- Test run details
- Rejudging tracking

### Redis (Queue & Cache)
- Judge job queue (priority queue)
- Submission status caching
- Real-time progress pub/sub
- Compiled binary cache

### MinIO / S3 / Local (Object Storage)
- Test case input/output files
- Submission source archives
- Compilation outputs
- Execution logs

```go
// Storage service configuration
type StorageConfig struct {
    // PostgreSQL
    DatabaseURL string `env:"DATABASE_URL"`
    
    // Redis
    RedisURL    string `env:"REDIS_URL"`
    
    // Object Storage
    StorageType string `env:"STORAGE_TYPE"` // "minio", "s3", "local"
    
    // MinIO/S3 config
    S3Endpoint  string `env:"S3_ENDPOINT"`
    S3Bucket    string `env:"S3_BUCKET"`
    S3AccessKey string `env:"S3_ACCESS_KEY"`
    S3SecretKey string `env:"S3_SECRET_KEY"`
    S3Region    string `env:"S3_REGION"`
    
    // Local storage config
    LocalPath   string `env:"LOCAL_STORAGE_PATH"`
}

// Object storage paths
const (
    // Testcase storage
    PathTestcaseInput  = "testcases/{problem_id}/{testcase_id}/input"
    PathTestcaseOutput = "testcases/{problem_id}/{testcase_id}/output"
    
    // Submission storage
    PathSubmissionSource = "submissions/{submission_id}/source"
    PathSubmissionBinary = "submissions/{submission_id}/binary"
    
    // Judging logs
    PathJudgingStdout = "judgings/{judging_id}/stdout"
    PathJudgingStderr = "judgings/{judging_id}/stderr"
    PathJudgingDiff   = "judgings/{judging_id}/diff"
)
```

### Storage Service Implementation

```go
// pkg/storage/storage.go
package storage

import (
    "context"
    "io"
    
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageService struct {
    client *minio.Client
    bucket string
    local  bool
    localPath string
}

func NewStorageService(cfg *StorageConfig) (*StorageService, error) {
    if cfg.StorageType == "local" {
        return &StorageService{
            local: true,
            localPath: cfg.LocalPath,
        }, nil
    }
    
    client, err := minio.New(cfg.S3Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
        Region: cfg.S3Region,
        Secure: true,
    })
    if err != nil {
        return nil, err
    }
    
    return &StorageService{
        client: client,
        bucket: cfg.S3Bucket,
        local: false,
    }, nil
}

func (s *StorageService) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
    if s.local {
        return s.uploadLocal(path, reader, size)
    }
    _, err := s.client.PutObject(ctx, s.bucket, path, reader, size, minio.PutObjectOptions{})
    return err
}

func (s *StorageService) Download(ctx context.Context, path string) (io.ReadCloser, error) {
    if s.local {
        return s.downloadLocal(path)
    }
    return s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
}
```

### Redis Queue Manager

```go
// pkg/queue/queue.go
package queue

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/redis/go-redis/v9"
)

type JudgeQueue struct {
    client *redis.Client
}

type JudgeJob struct {
    SubmissionID   string    `json:"submission_id"`
    ProblemID      string    `json:"problem_id"`
    Language       string    `json:"language"`
    Priority       int       `json:"priority"`
    TeamPending    int       `json:"team_pending"`
    SubmitTime     time.Time `json:"submit_time"`
    ContestID      string    `json:"contest_id,omitempty"`
}

// Push job to priority queue
func (q *JudgeQueue) Push(ctx context.Context, job *JudgeJob) error {
    data, _ := json.Marshal(job)
    
    // Use Redis sorted set for priority queue
    // Lower score = higher priority
    score := float64(job.Priority*1000) - float64(job.SubmitTime.Unix())
    
    return q.client.ZAdd(ctx, "judge:queue", &redis.Z{
        Score:  score,
        Member: data,
    }).Err()
}

// Pop highest priority job
func (q *JudgeQueue) Pop(ctx context.Context) (*JudgeJob, error) {
    // Get lowest score (highest priority)
    result, err := q.client.ZPopMin(ctx, "judge:queue").Result()
    if err != nil {
        return nil, err
    }
    
    var job JudgeJob
    json.Unmarshal([]byte(result[0].Member.(string)), &job)
    return &job, nil
}

// Get queue length
func (q *JudgeQueue) Length(ctx context.Context) (int64, error) {
    return q.client.ZCard(ctx, "judge:queue").Result()
}
```

### Redis Pub/Sub for Progress

```go
// Publish judging progress
func (q *JudgeQueue) PublishProgress(ctx context.Context, submissionID string, progress *JudgeProgress) error {
    data, _ := json.Marshal(progress)
    return q.client.Publish(ctx, fmt.Sprintf("progress:%s", submissionID), data).Err()
}

// Subscribe to progress updates
func (q *JudgeQueue) SubscribeProgress(ctx context.Context, submissionID string) <-chan *JudgeProgress {
    ch := make(chan *JudgeProgress)
    
    pubsub := q.client.Subscribe(ctx, fmt.Sprintf("progress:%s", submissionID))
    
    go func() {
        defer close(ch)
        for msg := range pubsub.Channel() {
            var progress JudgeProgress
            json.Unmarshal([]byte(msg.Payload), &progress)
            ch <- &progress
        }
    }()
    
    return ch
}
```

## Judgehost Configuration

```yaml
# judgehost config
judgehost:
  id: "judgehost-01"
  orchestrator_url: "http://domserver:8080/api"
  
  # Number of parallel judgedaemons (one per CPU core recommended)
  daemons: 4
  
  # Work directory
  workdir: "/var/lib/judgehost/work"
  
  # Chroot environment
  chroot_dir: "/var/lib/judgehost/chroot"
  
  # Cleanup settings
  disk_space_error: 1073741824  # 1GB minimum free
  auto_cleanup: true
  
  # Cgroup settings
  cgroup:
    memory: true
    cpuset: true
    
  # Logging
  log_level: "info"
  log_file: "/var/log/judgehost/judgedaemon.log"
```

## Database Schema (PostgreSQL - DOMjudge style)

### Submissions Table
```sql
CREATE TABLE submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    problem_id UUID NOT NULL,
    contest_id UUID,
    language_id VARCHAR(50) NOT NULL,
    source_code TEXT NOT NULL,
    submit_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- For team priority calculation
    team_pending_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_submissions_user ON submissions(user_id);
CREATE INDEX idx_submissions_problem ON submissions(problem_id);
CREATE INDEX idx_submissions_contest ON submissions(contest_id);
CREATE INDEX idx_submissions_time ON submissions(submit_time DESC);
```

### Judgings Table
```sql
CREATE TABLE judgings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID REFERENCES submissions(id),
    judgehost_id VARCHAR(100) NOT NULL,
    
    -- Timing
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    
    -- Results
    verdict VARCHAR(20),
    max_runtime DECIMAL(10, 3),  -- seconds
    max_memory INTEGER,           -- kilobytes
    
    -- Compilation
    compile_success BOOLEAN DEFAULT false,
    compile_output TEXT,
    compile_metadata JSONB,
    
    -- Verification
    valid BOOLEAN DEFAULT true,       -- False if rejudged
    verified BOOLEAN DEFAULT false,
    verified_by VARCHAR(100),
    verify_comment TEXT,
    seen BOOLEAN DEFAULT false,       -- Team has seen result
    
    -- For caching
    uuid UUID DEFAULT uuid_generate_v4(),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_judgings_submission ON judgings(submission_id);
CREATE INDEX idx_judgings_valid ON judgings(valid);
CREATE INDEX idx_judgings_verdict ON judgings(verdict);
```

### Judging Runs Table (per-testcase results)
```sql
CREATE TABLE judging_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    judging_id UUID REFERENCES judgings(id),
    testcase_id UUID REFERENCES testcases(id),
    rank INTEGER NOT NULL,
    
    -- Timing
    runtime DECIMAL(10, 3),     -- seconds
    wall_time DECIMAL(10, 3),
    memory INTEGER,              -- kilobytes
    
    -- Result
    verdict VARCHAR(20) NOT NULL,
    
    -- Output
    output_run TEXT,            -- Program stdout
    output_diff TEXT,           -- Diff with expected
    output_error TEXT,          -- Program stderr
    output_system TEXT,         -- System messages
    
    -- Metadata
    metadata JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_judging_runs_judging ON judging_runs(judging_id);
CREATE INDEX idx_judging_runs_testcase ON judging_runs(testcase_id);
```

## Monitoring & Metrics

```go
var (
    // DOMjudge-style metrics
    submissionsJudged = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "judge_submissions_judged_total",
            Help: "Total submissions judged",
        },
        []string{"verdict", "language"},
    )
    
    judgeDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "judge_duration_seconds",
            Help: "Time to judge a submission",
            Buckets: []float64{.1, .5, 1, 2, 5, 10, 30, 60, 120},
        },
        []string{"language"},
    )
    
    testcasesRun = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "judge_testcases_run_total",
            Help: "Total testcases executed",
        },
        []string{"problem_id"},
    )
    
    queueLength = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "judge_queue_length",
            Help: "Current judge queue length",
        },
    )
    
    judgehostActive = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "judgehost_active_judgings",
            Help: "Number of active judgings per judgehost",
        },
        []string{"judgehost_id"},
    )
)
```

## Security Measures

Following DOMjudge's security model:

1. **cgroups**: Memory, CPU, and process limiting via Linux cgroups
2. **chroot**: Isolated filesystem environment
3. **User isolation**: Run submissions as unprivileged user
4. **Network isolation**: No network access during execution
5. **Filesystem isolation**: Read-only system + writable temp
6. **Process limits**: Prevent fork bombs
7. **Output limits**: Prevent excessive output

## Next Steps

1. Set up Go project structure
2. Implement judgedaemon with cgroup integration
3. Build compile module with caching
4. Create runguard-style execution wrapper
5. Implement default and special validators
6. Set up Redis queue integration
7. Add WebSocket progress updates
8. Implement verification workflow
9. Deploy judgehost pool
10. Add monitoring and alerting