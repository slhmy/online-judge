package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	conteststore "github.com/slhmy/online-judge/backend/internal/contest/store"
	"github.com/slhmy/online-judge/backend/internal/judge/store"
	"github.com/slhmy/online-judge/backend/internal/pkg/middleware"
	"github.com/slhmy/online-judge/backend/internal/queue"
	submissionstore "github.com/slhmy/online-judge/backend/internal/submission/store"
	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/judge/v1"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrRejudgeNotFound = errors.New("rejudge not found")
var ErrInvalidStatus = errors.New("invalid rejudge status for this operation")
var ErrRejudgeInProgress = errors.New("rejudge is still in progress")

// JudgeService implements the JudgeService gRPC service
type JudgeService struct {
	pb.UnimplementedJudgeServiceServer
	store           store.JudgehostStoreInterface
	queue           *queue.AsynqClient
	rejudgeStore    store.RejudgeStoreInterface
	submissionStore submissionstore.SubmissionStoreInterface
	contestStore    conteststore.ContestStoreInterface
	redis           *redis.Client
}

// NewJudgeService creates a new judge service
func NewJudgeService(
	s store.JudgehostStoreInterface,
	asynqClient *queue.AsynqClient,
	rejudgeStore store.RejudgeStoreInterface,
	subStore submissionstore.SubmissionStoreInterface,
	contestStore conteststore.ContestStoreInterface,
	redisClient *redis.Client,
) *JudgeService {
	return &JudgeService{
		store:           s,
		queue:           asynqClient,
		rejudgeStore:    rejudgeStore,
		submissionStore: subStore,
		contestStore:    contestStore,
		redis:           redisClient,
	}
}

// RegisterJudgehost registers a new judgehost and returns the assigned queue name
func (s *JudgeService) RegisterJudgehost(ctx context.Context, req *pb.RegisterJudgehostRequest) (*pb.RegisterJudgehostResponse, error) {
	// Extract capabilities from request
	caps := req.Capabilities
	if caps == nil {
		caps = &pb.JudgehostCapabilities{}
	}

	// Register in store
	queueName, err := s.store.Register(ctx, req.JudgehostId,
		caps.Languages,
		caps.MaxConcurrentJobs,
		caps.MemoryLimit,
		caps.TimeLimit,
		caps.SupportsInteractive,
		caps.SupportsSpecial,
		caps.Extra,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register judgehost: %w", err)
	}

	return &pb.RegisterJudgehostResponse{
		JudgehostId: req.JudgehostId,
		QueueName:   queueName,
		Status:      pb.JudgehostStatus_JUDGEHOST_STATUS_IDLE,
	}, nil
}

// Heartbeat updates judgehost status and returns pending tasks
func (s *JudgeService) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	// Convert status to string
	statusStr := mapProtoStatusToString(req.Status)

	// Update heartbeat in store
	err := s.store.UpdateHeartbeat(ctx, req.JudgehostId, statusStr, req.CurrentJobId, req.ActiveJobs, req.CompletedJobIds)
	if err != nil {
		return nil, fmt.Errorf("failed to update heartbeat: %w", err)
	}

	// Get pending tasks count (would query Asynq queue stats)
	queueName := fmt.Sprintf("judgehost-%s", req.JudgehostId)
	pendingCount, _ := s.store.GetPendingTasksCount(ctx, queueName)

	// If judgehost is idle, return a small preview of pending task IDs as assignment hints.
	assignedTasks := []string{}
	if req.Status == pb.JudgehostStatus_JUDGEHOST_STATUS_IDLE && req.ActiveJobs == 0 && pendingCount > 0 {
		if ids, err := s.store.PeekPendingTaskIDs(ctx, queueName, 3); err == nil {
			assignedTasks = ids
		}
	}

	return &pb.HeartbeatResponse{
		Acknowledged:    true,
		PendingTasks:    pendingCount,
		AssignedTaskIds: assignedTasks,
	}, nil
}

// AssignTask assigns a submission to a judgehost based on load balancing strategy
func (s *JudgeService) AssignTask(ctx context.Context, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error) {
	// Get available judgehosts matching the requirements
	judgehosts, err := s.store.GetAvailableJudgehosts(ctx, req.LanguageId, req.Interactive, req.Special)
	if err != nil {
		return nil, fmt.Errorf("failed to get available judgehosts: %w", err)
	}

	// Load balancing strategy: prefer idle + recently active
	var selectedJudgehost *store.Judgehost
	if len(judgehosts) > 0 {
		// Select the first available judgehost (they're sorted by last_ping DESC)
		// This prefers idle hosts with recent activity
		selectedJudgehost = judgehosts[0]
	}

	// Enqueue task to appropriate queue
	queueName := "default"
	judgehostID := ""
	assigned := false

	if selectedJudgehost != nil {
		queueName = selectedJudgehost.QueueName
		judgehostID = selectedJudgehost.ID
		assigned = true

		// Update judgehost status
		_ = s.store.IncrementActiveJobs(ctx, judgehostID)
	}

	// Enqueue to Asynq
	payload := &queue.JudgeJobPayload{
		SubmissionID: req.SubmissionId,
		ProblemID:    req.ProblemId,
		Language:     req.LanguageId,
		Priority:     int(req.Priority),
	}

	_, err = s.queue.EnqueueJudgeTaskToQueue(payload, queueName)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	return &pb.AssignTaskResponse{
		SubmissionId: req.SubmissionId,
		JudgehostId:  judgehostID,
		QueueName:    queueName,
		Assigned:     assigned,
	}, nil
}

// ListJudgehosts lists all registered judgehosts
func (s *JudgeService) ListJudgehosts(ctx context.Context, req *pb.ListJudgehostsRequest) (*pb.ListJudgehostsResponse, error) {
	statusFilter := mapProtoStatusToString(req.StatusFilter)

	judgehosts, total, err := s.store.List(ctx, statusFilter, req.Page, req.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list judgehosts: %w", err)
	}

	var pbJudgehosts []*pb.Judgehost
	for _, j := range judgehosts {
		pbJudgehosts = append(pbJudgehosts, &pb.Judgehost{
			Id:            j.ID,
			QueueName:     j.QueueName,
			Capabilities:  convertCapabilities(j),
			Status:        mapStringStatusToProto(j.Status),
			CurrentJobId:  j.CurrentJobID,
			LastPing:      j.LastPing.Format(time.RFC3339),
			ActiveJobs:    j.ActiveJobs,
			CompletedJobs: j.CompletedJobs,
			RegisteredAt:  j.RegisteredAt.Format(time.RFC3339),
		})
	}

	return &pb.ListJudgehostsResponse{
		Judgehosts: pbJudgehosts,
		Total:      total,
	}, nil
}

// GetJudgehost retrieves a specific judgehost
func (s *JudgeService) GetJudgehost(ctx context.Context, req *pb.GetJudgehostRequest) (*pb.GetJudgehostResponse, error) {
	j, err := s.store.GetByID(ctx, req.JudgehostId)
	if err != nil {
		return nil, fmt.Errorf("failed to get judgehost: %w", err)
	}

	return &pb.GetJudgehostResponse{
		Judgehost: &pb.Judgehost{
			Id:            j.ID,
			QueueName:     j.QueueName,
			Capabilities:  convertCapabilities(j),
			Status:        mapStringStatusToProto(j.Status),
			CurrentJobId:  j.CurrentJobID,
			LastPing:      j.LastPing.Format(time.RFC3339),
			ActiveJobs:    j.ActiveJobs,
			CompletedJobs: j.CompletedJobs,
			RegisteredAt:  j.RegisteredAt.Format(time.RFC3339),
		},
	}, nil
}

// DeregisterJudgehost removes a judgehost from the system
func (s *JudgeService) DeregisterJudgehost(ctx context.Context, req *pb.DeregisterJudgehostRequest) (*pb.DeregisterJudgehostResponse, error) {
	reassignedTasks, err := s.store.Deregister(ctx, req.JudgehostId, req.Force)
	if err != nil {
		return nil, fmt.Errorf("failed to deregister judgehost: %w", err)
	}

	return &pb.DeregisterJudgehostResponse{
		JudgehostId:     req.JudgehostId,
		Success:         true,
		ReassignedTasks: reassignedTasks,
	}, nil
}

// ============= Rejudge methods =============

// CreateRejudge creates a new rejudging operation
func (s *JudgeService) CreateRejudge(ctx context.Context, req *pb.CreateRejudgeRequest) (*pb.CreateRejudgeResponse, error) {
	// Get user ID from context (admin only)
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, ErrUnauthorized
	}

	// Get submissions to rejudge
	submissionIDs, err := s.rejudgeStore.GetSubmissionsForRejudge(ctx, req.ContestId, req.ProblemId, req.FromVerdict, req.SubmissionIds)
	if err != nil {
		return nil, err
	}

	if len(submissionIDs) == 0 {
		return nil, fmt.Errorf("no submissions found matching criteria")
	}

	// Create rejudge record
	rejudgeID, affectedCount, err := s.rejudgeStore.CreateRejudge(ctx, userID, req.ContestId, req.ProblemId, req.FromVerdict, req.Reason, submissionIDs)
	if err != nil {
		return nil, err
	}

	// Create rejudging_submission records and queue submissions
	for _, submissionID := range submissionIDs {
		// Get current judging
		judging, err := s.submissionStore.GetJudging(ctx, submissionID)
		if err != nil {
			// Submission might not have been judged yet, skip
			continue
		}

		// Create rejudging_submission record
		err = s.rejudgeStore.CreateRejudgeSubmission(ctx, rejudgeID, submissionID, judging.ID, judging.Verdict)
		if err != nil {
			continue
		}

		// Get submission details
		sub, err := s.submissionStore.GetByID(ctx, submissionID)
		if err != nil {
			continue
		}

		// Queue submission for rejudging
		job := RejudgeJob{
			SubmissionID: submissionID,
			ProblemID:    sub.ProblemID,
			Language:     sub.LanguageID,
			Priority:     20, // Higher priority for rejudges
			SubmitTime:   time.Now(),
			RejudgeID:    rejudgeID,
		}

		if err := s.pushToQueue(ctx, &job); err != nil {
			// Log but continue
			fmt.Printf("Failed to queue submission %s for rejudge: %v\n", submissionID, err)
		}
	}

	// Update status to judging
	if err := s.rejudgeStore.UpdateRejudgeStatus(ctx, rejudgeID, "judging"); err != nil {
		// Log but continue - status update is not critical
		fmt.Printf("Failed to update rejudge status: %v\n", err)
	}

	return &pb.CreateRejudgeResponse{
		Id:            rejudgeID,
		AffectedCount: affectedCount,
		Status:        pb.RejudgeStatus_REJUDGE_STATUS_JUDGING,
	}, nil
}

// GetRejudge retrieves a rejudging operation
func (s *JudgeService) GetRejudge(ctx context.Context, req *pb.GetRejudgeRequest) (*pb.GetRejudgeResponse, error) {
	rejudge, err := s.rejudgeStore.GetRejudge(ctx, req.Id)
	if err != nil {
		return nil, ErrRejudgeNotFound
	}

	// Get progress
	pending, judging, done, err := s.rejudgeStore.GetRejudgeProgress(ctx, req.Id)
	if err != nil {
		pending = 0
		judging = 0
		done = 0
	}

	return &pb.GetRejudgeResponse{
		Rejudge: &pb.Rejudge{
			Id:            rejudge.ID,
			UserId:        rejudge.UserID,
			ContestId:     rejudge.ContestID,
			ProblemId:     rejudge.ProblemID,
			Status:        mapRejudgeStatusToProto(rejudge.Status),
			Reason:        rejudge.Reason,
			AffectedCount: rejudge.AffectedCount,
			JudgedCount:   done,
			PendingCount:  pending + judging,
			CreatedAt:     timestamppb.New(rejudge.CreatedAt),
			StartedAt:     timestamppb.New(rejudge.StartedAt),
			FinishedAt:    timestamppb.New(rejudge.FinishedAt),
			AppliedAt:     timestamppb.New(rejudge.AppliedAt),
			RevertedAt:    timestamppb.New(rejudge.RevertedAt),
		},
	}, nil
}

// ListRejudges lists rejudging operations
func (s *JudgeService) ListRejudges(ctx context.Context, req *pb.ListRejudgesRequest) (*pb.ListRejudgesResponse, error) {
	page := req.GetPagination().GetPage()
	pageSize := req.GetPagination().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	rejudges, total, err := s.rejudgeStore.ListRejudges(ctx, req.ContestId, req.ProblemId, mapRejudgeStatusFromProto(req.Status), req.UserId, page, pageSize)
	if err != nil {
		return nil, err
	}

	var pbRejudges []*pb.Rejudge
	for _, r := range rejudges {
		pbRejudges = append(pbRejudges, &pb.Rejudge{
			Id:            r.ID,
			UserId:        r.UserID,
			ContestId:     r.ContestID,
			ProblemId:     r.ProblemID,
			Status:        mapRejudgeStatusToProto(r.Status),
			Reason:        r.Reason,
			AffectedCount: r.AffectedCount,
			CreatedAt:     timestamppb.New(r.CreatedAt),
			StartedAt:     timestamppb.New(r.StartedAt),
			FinishedAt:    timestamppb.New(r.FinishedAt),
			AppliedAt:     timestamppb.New(r.AppliedAt),
			RevertedAt:    timestamppb.New(r.RevertedAt),
		})
	}

	return &pb.ListRejudgesResponse{
		Rejudges: pbRejudges,
		Pagination: &commonv1.PaginatedResponse{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	}, nil
}

// CancelRejudge cancels a pending rejudging operation
func (s *JudgeService) CancelRejudge(ctx context.Context, req *pb.CancelRejudgeRequest) (*pb.CancelRejudgeResponse, error) {
	rejudge, err := s.rejudgeStore.GetRejudge(ctx, req.Id)
	if err != nil {
		return nil, ErrRejudgeNotFound
	}

	// Can only cancel pending or judging rejudges
	if rejudge.Status != "pending" && rejudge.Status != "judging" {
		return nil, ErrInvalidStatus
	}

	err = s.rejudgeStore.CancelRejudge(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.CancelRejudgeResponse{
		Id:     req.Id,
		Status: pb.RejudgeStatus_REJUDGE_STATUS_CANCELLED,
	}, nil
}

// ApplyRejudge applies the rejudge results
func (s *JudgeService) ApplyRejudge(ctx context.Context, req *pb.ApplyRejudgeRequest) (*pb.ApplyRejudgeResponse, error) {
	rejudge, err := s.rejudgeStore.GetRejudge(ctx, req.Id)
	if err != nil {
		return nil, ErrRejudgeNotFound
	}

	// Can only apply judged rejudges
	if rejudge.Status != "judged" {
		if rejudge.Status == "judging" {
			return nil, ErrRejudgeInProgress
		}
		return nil, ErrInvalidStatus
	}

	verdictChanges, err := s.rejudgeStore.ApplyRejudge(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// Update scoreboard if this is a contest rejudge
	if rejudge.ContestID != "" && verdictChanges > 0 {
		err = s.contestStore.RefreshScoreboard(ctx, rejudge.ContestID)
		if err != nil {
			// Log but don't fail
			fmt.Printf("Failed to refresh scoreboard for contest %s: %v\n", rejudge.ContestID, err)
		}
	}

	return &pb.ApplyRejudgeResponse{
		Id:              req.Id,
		VerdictsChanged: verdictChanges,
		Status:          pb.RejudgeStatus_REJUDGE_STATUS_APPLIED,
	}, nil
}

// RevertRejudge reverts the rejudge results
func (s *JudgeService) RevertRejudge(ctx context.Context, req *pb.RevertRejudgeRequest) (*pb.RevertRejudgeResponse, error) {
	rejudge, err := s.rejudgeStore.GetRejudge(ctx, req.Id)
	if err != nil {
		return nil, ErrRejudgeNotFound
	}

	// Can only revert applied rejudges
	if rejudge.Status != "applied" {
		return nil, ErrInvalidStatus
	}

	restoredCount, err := s.rejudgeStore.RevertRejudge(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// Update scoreboard if this is a contest rejudge
	if rejudge.ContestID != "" {
		err = s.contestStore.RefreshScoreboard(ctx, rejudge.ContestID)
		if err != nil {
			fmt.Printf("Failed to refresh scoreboard for contest %s: %v\n", rejudge.ContestID, err)
		}
	}

	return &pb.RevertRejudgeResponse{
		Id:               req.Id,
		VerdictsRestored: restoredCount,
		Status:           pb.RejudgeStatus_REJUDGE_STATUS_REVERTED,
	}, nil
}

// GetRejudgeSubmissions retrieves submissions for a rejudging operation
func (s *JudgeService) GetRejudgeSubmissions(ctx context.Context, req *pb.GetRejudgeSubmissionsRequest) (*pb.GetRejudgeSubmissionsResponse, error) {
	page := req.GetPagination().GetPage()
	pageSize := req.GetPagination().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	status := ""
	if req.Status != pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_UNSPECIFIED {
		status = mapRejudgeSubmissionStatusFromProto(req.Status)
	}

	submissions, total, err := s.rejudgeStore.GetRejudgeSubmissions(ctx, req.RejudgeId, req.OnlyChanged, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	var pbSubmissions []*pb.RejudgeSubmission
	for _, rs := range submissions {
		pbSubmissions = append(pbSubmissions, &pb.RejudgeSubmission{
			SubmissionId:      rs.SubmissionID,
			OriginalJudgingId: rs.OriginalJudgingID,
			NewJudgingId:      rs.NewJudgingID,
			OriginalVerdict:   rs.OriginalVerdict,
			NewVerdict:        rs.NewVerdict,
			VerdictChanged:    rs.VerdictChanged,
			Status:            mapRejudgeSubmissionStatusToProto(rs.Status),
		})
	}

	return &pb.GetRejudgeSubmissionsResponse{
		Submissions: pbSubmissions,
		Pagination: &commonv1.PaginatedResponse{
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		},
	}, nil
}

// InternalUpdateRejudgeSubmission updates a submission's status after judging
func (s *JudgeService) InternalUpdateRejudgeSubmission(ctx context.Context, req *pb.InternalUpdateRejudgeSubmissionRequest) (*pb.InternalUpdateRejudgeSubmissionResponse, error) {
	err := s.rejudgeStore.UpdateRejudgeSubmission(ctx, req.RejudgeId, req.SubmissionId, req.NewJudgingId, req.NewVerdict)
	if err != nil {
		return nil, err
	}

	// Check if all submissions are done
	pending, judging, _, err := s.rejudgeStore.GetRejudgeProgress(ctx, req.RejudgeId)
	if err == nil && pending == 0 && judging == 0 {
		// All done, update rejudge status
		if err := s.rejudgeStore.UpdateRejudgeStatus(ctx, req.RejudgeId, "judged"); err != nil {
			// Log but continue - status update is not critical
			fmt.Printf("Failed to update rejudge status: %v\n", err)
		}
	}

	return &pb.InternalUpdateRejudgeSubmissionResponse{
		Status: "updated",
	}, nil
}

// ============= Helper functions =============

func mapProtoStatusToString(status pb.JudgehostStatus) string {
	switch status {
	case pb.JudgehostStatus_JUDGEHOST_STATUS_IDLE:
		return "idle"
	case pb.JudgehostStatus_JUDGEHOST_STATUS_BUSY:
		return "busy"
	case pb.JudgehostStatus_JUDGEHOST_STATUS_OFFLINE:
		return "offline"
	case pb.JudgehostStatus_JUDGEHOST_STATUS_ERROR:
		return "error"
	default:
		return "unspecified"
	}
}

func mapStringStatusToProto(status string) pb.JudgehostStatus {
	switch status {
	case "idle":
		return pb.JudgehostStatus_JUDGEHOST_STATUS_IDLE
	case "busy":
		return pb.JudgehostStatus_JUDGEHOST_STATUS_BUSY
	case "offline":
		return pb.JudgehostStatus_JUDGEHOST_STATUS_OFFLINE
	case "error":
		return pb.JudgehostStatus_JUDGEHOST_STATUS_ERROR
	default:
		return pb.JudgehostStatus_JUDGEHOST_STATUS_UNSPECIFIED
	}
}

func convertCapabilities(j *store.Judgehost) *pb.JudgehostCapabilities {
	return &pb.JudgehostCapabilities{
		Languages:           j.Languages,
		MaxConcurrentJobs:   j.MaxConcurrent,
		MemoryLimit:         j.MemoryLimit,
		TimeLimit:           j.TimeLimit,
		SupportsInteractive: j.Interactive,
		SupportsSpecial:     j.Special,
		Extra:               j.Extra,
	}
}

// Rejudge status helpers

func mapRejudgeStatusToProto(status string) pb.RejudgeStatus {
	switch status {
	case "pending":
		return pb.RejudgeStatus_REJUDGE_STATUS_PENDING
	case "judging":
		return pb.RejudgeStatus_REJUDGE_STATUS_JUDGING
	case "judged":
		return pb.RejudgeStatus_REJUDGE_STATUS_JUDGED
	case "applied":
		return pb.RejudgeStatus_REJUDGE_STATUS_APPLIED
	case "reverted":
		return pb.RejudgeStatus_REJUDGE_STATUS_REVERTED
	case "cancelled":
		return pb.RejudgeStatus_REJUDGE_STATUS_CANCELLED
	default:
		return pb.RejudgeStatus_REJUDGE_STATUS_UNSPECIFIED
	}
}

func mapRejudgeStatusFromProto(status pb.RejudgeStatus) string {
	switch status {
	case pb.RejudgeStatus_REJUDGE_STATUS_PENDING:
		return "pending"
	case pb.RejudgeStatus_REJUDGE_STATUS_JUDGING:
		return "judging"
	case pb.RejudgeStatus_REJUDGE_STATUS_JUDGED:
		return "judged"
	case pb.RejudgeStatus_REJUDGE_STATUS_APPLIED:
		return "applied"
	case pb.RejudgeStatus_REJUDGE_STATUS_REVERTED:
		return "reverted"
	case pb.RejudgeStatus_REJUDGE_STATUS_CANCELLED:
		return "cancelled"
	default:
		return ""
	}
}

func mapRejudgeSubmissionStatusToProto(status string) pb.RejudgeSubmissionStatus {
	switch status {
	case "pending":
		return pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_PENDING
	case "judging":
		return pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_JUDGING
	case "done":
		return pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_DONE
	default:
		return pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_UNSPECIFIED
	}
}

func mapRejudgeSubmissionStatusFromProto(status pb.RejudgeSubmissionStatus) string {
	switch status {
	case pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_PENDING:
		return "pending"
	case pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_JUDGING:
		return "judging"
	case pb.RejudgeSubmissionStatus_REJUDGE_SUBMISSION_STATUS_DONE:
		return "done"
	default:
		return ""
	}
}

func (s *JudgeService) pushToQueue(ctx context.Context, job *RejudgeJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	score := float64(job.Priority*1000000) - float64(job.SubmitTime.Unix())

	return s.redis.ZAdd(ctx, "judge:queue", redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()
}

// RejudgeJob extended with rejudge_id
type RejudgeJob struct {
	SubmissionID string    `json:"submission_id"`
	ProblemID    string    `json:"problem_id"`
	Language     string    `json:"language"`
	Priority     int       `json:"priority"`
	SubmitTime   time.Time `json:"submit_time"`
	RejudgeID    string    `json:"rejudge_id,omitempty"`
}
