package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/submission/v1"
	"github.com/online-judge/backend/internal/queue"
	"github.com/online-judge/backend/internal/submission/store"
)

type SubmissionService struct {
	pb.UnimplementedSubmissionServiceServer
	store        store.SubmissionStoreInterface
	redis        *redis.Client
	queue        *queue.AsynqClient
	legacyQueue  *queue.LegacyQueue
	useAsynq     bool
	useLegacy    bool
}

func NewSubmissionService(s store.SubmissionStoreInterface, redis *redis.Client, asynqClient *queue.AsynqClient, legacyQueue *queue.LegacyQueue, useAsynq bool, useLegacy bool) *SubmissionService {
	return &SubmissionService{
		store:       s,
		redis:       redis,
		queue:       asynqClient,
		legacyQueue: legacyQueue,
		useAsynq:    useAsynq,
		useLegacy:   useLegacy,
	}
}

func (s *SubmissionService) CreateSubmission(ctx context.Context, req *pb.CreateSubmissionRequest) (*pb.CreateSubmissionResponse, error) {
	// TODO: Get user ID from context (from JWT)
	userID := "00000000-0000-0000-0000-000000000001"

	// Create submission record with source code
	id, err := s.store.Create(ctx, userID, req.ProblemId, req.ContestId, req.LanguageId, req.SourceCode)
	if err != nil {
		return nil, err
	}

	// Cache source code in Redis for judge daemon (24 hour TTL)
	sourceKey := "submission:" + id + ":source"
	s.redis.Set(ctx, sourceKey, req.SourceCode, 24*time.Hour)

	// Push to judge queue(s) - dual-write during migration
	payload := &queue.JudgeJobPayload{
		SubmissionID: id,
		ProblemID:    req.ProblemId,
		Language:     req.LanguageId,
		Priority:     0,
	}

	if err := s.enqueueJudgeJob(ctx, payload); err != nil {
		return nil, err
	}

	return &pb.CreateSubmissionResponse{
		Id:     id,
		Status: pb.SubmissionStatus_SUBMISSION_STATUS_QUEUED,
	}, nil
}

// enqueueJudgeJob handles dual-write to both asynq and legacy queue during migration
func (s *SubmissionService) enqueueJudgeJob(ctx context.Context, payload *queue.JudgeJobPayload) error {
	var asynqErr, legacyErr error

	// Enqueue to asynq queue (primary)
	if s.useAsynq && s.queue != nil {
		if _, err := s.queue.EnqueueJudgeTask(payload); err != nil {
			asynqErr = err
		}
	}

	// Enqueue to legacy queue (for migration)
	if s.useLegacy && s.legacyQueue != nil {
		if err := s.legacyQueue.EnqueueJudgeJob(ctx, payload); err != nil {
			legacyErr = err
		}
	}

	// Return error if all enabled queues failed
	if s.useAsynq && s.useLegacy {
		if asynqErr != nil && legacyErr != nil {
			return fmt.Errorf("both queues failed: asynq=%v, legacy=%v", asynqErr, legacyErr)
		}
		// If only one failed, log but don't error (at least one succeeded)
		if asynqErr != nil {
			// Log the error but don't fail - legacy queue succeeded
		}
		if legacyErr != nil {
			// Log the error but don't fail - asynq queue succeeded
		}
		return nil
	}

	// Single queue mode
	if s.useAsynq && asynqErr != nil {
		return asynqErr
	}
	if s.useLegacy && legacyErr != nil {
		return legacyErr
	}

	return nil
}

func (s *SubmissionService) GetSubmission(ctx context.Context, req *pb.GetSubmissionRequest) (*pb.GetSubmissionResponse, error) {
	sub, err := s.store.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// Get latest judging from database first, then fallback to Redis cache
	judging, err := s.store.GetJudging(ctx, req.Id)
	if err != nil && s.redis != nil {
		// Try Redis cache (from judge daemon)
		judgingKey := "judging:judging-" + req.Id + ":meta"
		result := s.redis.HGetAll(ctx, judgingKey)
		if result.Err() == nil && len(result.Val()) > 0 {
			data := result.Val()
			verdict := data["verdict"]
			if verdict == "" {
				verdict = data["status"]
			}

			maxRuntime := 0.0
			if v, ok := data["max_runtime"]; ok {
				_, _ = fmt.Sscanf(v, "%f", &maxRuntime)
			}

			maxMemory := int32(0)
			if v, ok := data["max_memory"]; ok {
				var m int64
				_, _ = fmt.Sscanf(v, "%d", &m)
				maxMemory = int32(m)
			}

			judging = &store.Judging{
				ID:           "judging-" + req.Id,
				SubmissionID: req.Id,
				JudgehostID:  data["judgehost_id"],
				Verdict:      verdict,
				MaxRuntime:   maxRuntime,
				MaxMemory:    maxMemory,
				Valid:        data["status"] == "completed",
			}
		} else {
			judging = nil
		}
	} else if err != nil {
		judging = nil
	}

	response := &pb.GetSubmissionResponse{
		Submission: &pb.Submission{
			Id:         sub.ID,
			UserId:     sub.UserID,
			ProblemId:  sub.ProblemID,
			ContestId:  sub.ContestID,
			LanguageId: sub.LanguageID,
			SourcePath: sub.SourceCode, // Return actual source code
			SubmitTime: sub.SubmitTime.Format(time.RFC3339),
		},
	}

	if judging != nil {
		response.LatestJudging = &pb.Judging{
			Id:             judging.ID,
			SubmissionId:   judging.SubmissionID,
			JudgehostId:    judging.JudgehostID,
			StartTime:      judging.StartTime.Format(time.RFC3339),
			EndTime:        judging.EndTime.Format(time.RFC3339),
			MaxRuntime:     judging.MaxRuntime,
			MaxMemory:      judging.MaxMemory,
			Verdict:        mapVerdictToProto(judging.Verdict),
			CompileSuccess: judging.CompileSuccess,
			Valid:          judging.Valid,
			Verified:       judging.Verified,
			VerifiedBy:     judging.VerifiedBy,
			Score:          judging.Score,
		}
	}

	return response, nil
}

func (s *SubmissionService) ListSubmissions(ctx context.Context, req *pb.ListSubmissionsRequest) (*pb.ListSubmissionsResponse, error) {
	subs, total, err := s.store.List(ctx, req.GetUserId(), req.GetProblemId(), req.GetContestId(), req.GetPagination().GetPage(), req.GetPagination().GetPageSize())
	if err != nil {
		return nil, err
	}

	var submissions []*pb.SubmissionSummary
	for _, sub := range subs {
		// Get judging verdict for this submission
		verdict := pb.Verdict_VERDICT_UNSPECIFIED
		runtime := 0.0
		memory := int32(0)

		// Try database first
		judging, err := s.store.GetJudging(ctx, sub.ID)
		if err == nil && judging != nil {
			verdict = mapVerdictToProto(judging.Verdict)
			runtime = judging.MaxRuntime
			memory = judging.MaxMemory
		} else if s.redis != nil {
			// Fallback to Redis cache
			judgingKey := "judging:judging-" + sub.ID + ":meta"
			result := s.redis.HGetAll(ctx, judgingKey)
			if result.Err() == nil && len(result.Val()) > 0 {
				data := result.Val()
				v := data["verdict"]
				if v == "" {
					v = data["status"]
				}
				verdict = mapVerdictToProto(v)

				if v, ok := data["max_runtime"]; ok {
					_, _ = fmt.Sscanf(v, "%f", &runtime)
				}
				if v, ok := data["max_memory"]; ok {
					var m int64
					_, _ = fmt.Sscanf(v, "%d", &m)
					memory = int32(m)
				}
			}
		}

		submissions = append(submissions, &pb.SubmissionSummary{
			Id:          sub.ID,
			ProblemId:   sub.ProblemID,
			ProblemName: sub.ProblemName,
			User: &commonv1.UserRef{
				Id: sub.UserID,
			},
			LanguageId: sub.LanguageID,
			SubmitTime: sub.SubmitTime.Format(time.RFC3339),
			Verdict:    verdict,
			Runtime:    runtime,
			Memory:     memory,
		})
	}

	return &pb.ListSubmissionsResponse{
		Submissions: submissions,
		Pagination: &commonv1.PaginatedResponse{
			Total:    total,
			Page:     req.GetPagination().GetPage(),
			PageSize: req.GetPagination().GetPageSize(),
		},
	}, nil
}

// GetJudging retrieves the judging result for a submission
// If judgingID is empty, returns the latest (most recent) valid judging
func (s *SubmissionService) GetJudging(ctx context.Context, req *pb.GetJudgingRequest) (*pb.GetJudgingResponse, error) {
	var judging *store.Judging
	var err error

	if req.JudgingId != "" {
		// Get specific judging by ID
		judging, err = s.store.GetJudgingByID(ctx, req.JudgingId)
	} else {
		// Get latest judging for submission
		judging, err = s.store.GetJudging(ctx, req.SubmissionId)
	}

	// If not found in database, try Redis cache (from judge daemon)
	if err != nil {
		if s.redis == nil {
			return nil, err
		}
		judgingKey := "judging:judging-" + req.SubmissionId + ":meta"
		result := s.redis.HGetAll(ctx, judgingKey)
		if result.Err() != nil || len(result.Val()) == 0 {
			return nil, err
		}

		data := result.Val()
		verdict := data["verdict"]
		if verdict == "" {
			verdict = data["status"]
		}

		maxRuntime := 0.0
		if v, ok := data["max_runtime"]; ok {
			_, _ = fmt.Sscanf(v, "%f", &maxRuntime)
		}

		maxMemory := int32(0)
		if v, ok := data["max_memory"]; ok {
			var m int64
			_, _ = fmt.Sscanf(v, "%d", &m)
			maxMemory = int32(m)
		}

		return &pb.GetJudgingResponse{
			Judging: &pb.Judging{
				Id:           "judging-" + req.SubmissionId,
				SubmissionId: req.SubmissionId,
				JudgehostId:  data["judgehost_id"],
				Verdict:      mapVerdictToProto(verdict),
				MaxRuntime:   maxRuntime,
				MaxMemory:    maxMemory,
				Valid:        data["status"] == "completed",
			},
		}, nil
	}

	return &pb.GetJudgingResponse{
		Judging: &pb.Judging{
			Id:             judging.ID,
			SubmissionId:   judging.SubmissionID,
			JudgehostId:    judging.JudgehostID,
			StartTime:      judging.StartTime.Format(time.RFC3339),
			EndTime:        judging.EndTime.Format(time.RFC3339),
			MaxRuntime:     judging.MaxRuntime,
			MaxMemory:      judging.MaxMemory,
			Verdict:        mapVerdictToProto(judging.Verdict),
			CompileSuccess: judging.CompileSuccess,
			Valid:          judging.Valid,
			Verified:       judging.Verified,
			VerifiedBy:     judging.VerifiedBy,
			Score:          judging.Score,
		},
	}, nil
}

func (s *SubmissionService) GetJudgingRuns(ctx context.Context, req *pb.GetJudgingRunsRequest) (*pb.GetJudgingRunsResponse, error) {
	var judgingID string

	if req.JudgingId != "" {
		judgingID = req.JudgingId
	} else {
		// Get latest judging for submission first
		judging, err := s.store.GetJudging(ctx, req.SubmissionId)
		if err != nil {
			return nil, err
		}
		judgingID = judging.ID
	}

	runs, err := s.store.GetJudgingRuns(ctx, judgingID)
	if err != nil {
		return nil, err
	}

	var pbRuns []*pb.JudgingRun
	for _, run := range runs {
		pbRuns = append(pbRuns, &pb.JudgingRun{
			Id:              run.ID,
			JudgingId:       run.JudgingID,
			TestCaseId:      run.TestCaseID,
			Rank:            run.Rank,
			Runtime:         run.Runtime,
			WallTime:        run.WallTime,
			Memory:          run.Memory,
			Verdict:         mapVerdictToProto(run.Verdict),
			OutputRunPath:   run.OutputRunPath,
			OutputDiffPath:  run.OutputDiffPath,
			OutputErrorPath: run.OutputErrorPath,
		})
	}

	return &pb.GetJudgingRunsResponse{
		Runs: pbRuns,
	}, nil
}

// GetSourceCode retrieves source code for a submission (internal use)
func (s *SubmissionService) GetSourceCode(ctx context.Context, submissionID string) (string, error) {
	// Try Redis cache first
	sourceKey := "submission:" + submissionID + ":source"
	source, err := s.redis.Get(ctx, sourceKey).Result()
	if err == nil {
		return source, nil
	}

	// Fall back to database
	sub, err := s.store.GetByID(ctx, submissionID)
	if err != nil {
		return "", err
	}

	// Re-cache in Redis
	s.redis.Set(ctx, sourceKey, sub.SourceCode, 24*time.Hour)

	return sub.SourceCode, nil
}

// mapVerdictToProto converts a database verdict string to proto enum
func mapVerdictToProto(verdict string) pb.Verdict {
	switch verdict {
	case "correct":
		return pb.Verdict_VERDICT_CORRECT
	case "wrong-answer":
		return pb.Verdict_VERDICT_WRONG_ANSWER
	case "timelimit", "time-limit":
		return pb.Verdict_VERDICT_TIMELIMIT
	case "memory-limit":
		return pb.Verdict_VERDICT_MEMORY_LIMIT
	case "run-error":
		return pb.Verdict_VERDICT_RUN_ERROR
	case "compiler-error":
		return pb.Verdict_VERDICT_COMPILER_ERROR
	case "output-limit":
		return pb.Verdict_VERDICT_OUTPUT_LIMIT
	case "presentation":
		return pb.Verdict_VERDICT_PRESENTATION
	default:
		return pb.Verdict_VERDICT_UNSPECIFIED
	}
}

// RejudgeSubmission re-runs the judging for a submission
func (s *SubmissionService) InternalCreateJudging(ctx context.Context, req *pb.InternalCreateJudgingRequest) (*pb.InternalCreateJudgingResponse, error) {
	judgingID, err := s.store.CreateJudging(ctx, req.SubmissionId, req.JudgehostId)
	if err != nil {
		return nil, err
	}

	// Also cache in Redis for fast lookup
	if s.redis != nil {
		judgingKey := "judging:judging-" + req.SubmissionId + ":meta"
		s.redis.HSet(ctx, judgingKey, map[string]interface{}{
			"submission_id": req.SubmissionId,
			"judgehost_id":  req.JudgehostId,
			"judging_id":    judgingID,
			"status":        "judging",
		})
		s.redis.Expire(ctx, judgingKey, 24*time.Hour)
	}

	return &pb.InternalCreateJudgingResponse{
		JudgingId: judgingID,
	}, nil
}

// InternalUpdateJudging updates a judging record with results
func (s *SubmissionService) InternalUpdateJudging(ctx context.Context, req *pb.InternalUpdateJudgingRequest) (*pb.InternalUpdateJudgingResponse, error) {
	// Determine compile_success: true if verdict is NOT compiler-error
	compileSuccess := req.Verdict != "compiler-error"

	err := s.store.UpdateJudging(ctx, req.JudgingId, req.Verdict, req.MaxRuntime, int64(req.MaxMemory), compileSuccess)
	if err != nil {
		return nil, err
	}

	// Also update Redis cache
	if s.redis != nil {
		judgingKey := "judging:" + req.JudgingId + ":meta"
		s.redis.HSet(ctx, judgingKey, map[string]interface{}{
			"verdict":         req.Verdict,
			"max_runtime":     req.MaxRuntime,
			"max_memory":      req.MaxMemory,
			"compile_success": compileSuccess,
			"status":          "completed",
		})
	}

	return &pb.InternalUpdateJudgingResponse{
		JudgingId: req.JudgingId,
		Status:    "updated",
	}, nil
}

// InternalCreateJudgingRun creates a single test case run record
func (s *SubmissionService) InternalCreateJudgingRun(ctx context.Context, req *pb.InternalCreateJudgingRunRequest) (*pb.InternalCreateJudgingRunResponse, error) {
	err := s.store.CreateJudgingRun(ctx, req.JudgingId, req.TestCaseId, int(req.Rank), req.Verdict, req.Runtime, req.WallTime, int64(req.Memory))
	if err != nil {
		return nil, err
	}

	return &pb.InternalCreateJudgingRunResponse{
		Status: "created",
	}, nil
}

// RejudgeSubmission re-runs the judging for a submission
func (s *SubmissionService) RejudgeSubmission(ctx context.Context, req *pb.RejudgeSubmissionRequest) (*pb.RejudgeSubmissionResponse, error) {
	// Get the submission
	sub, err := s.store.GetByID(ctx, req.SubmissionId)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	// Invalidate the previous judging (mark as invalid)
	judging, err := s.store.GetJudging(ctx, req.SubmissionId)
	if err == nil && judging != nil {
		_ = s.store.InvalidateJudging(ctx, judging.ID)
	}

	// Clear Redis cache for this submission's judging
	judgingKey := "judging:judging-" + req.SubmissionId + ":meta"
	_ = s.redis.Del(ctx, judgingKey)

	// Push to judge queue(s) - dual-write during migration
	payload := &queue.JudgeJobPayload{
		SubmissionID: req.SubmissionId,
		ProblemID:    sub.ProblemID,
		Language:     sub.LanguageID,
		Priority:     10, // Higher priority for rejudges
	}

	if err := s.enqueueJudgeJob(ctx, payload); err != nil {
		return nil, fmt.Errorf("failed to queue rejudge: %w", err)
	}

	// Re-cache source code if not already cached
	sourceKey := "submission:" + req.SubmissionId + ":source"
	s.redis.Set(ctx, sourceKey, sub.SourceCode, 24*time.Hour)

	return &pb.RejudgeSubmissionResponse{
		SubmissionId: req.SubmissionId,
		Status:       "queued",
	}, nil
}
