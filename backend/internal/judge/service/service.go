package service

import (
	"context"
	"fmt"
	"time"

	pb "github.com/online-judge/backend/gen/go/judge/v1"
	"github.com/online-judge/backend/internal/judge/store"
	"github.com/online-judge/backend/internal/queue"
)

// JudgeService implements the JudgeService gRPC service
type JudgeService struct {
	pb.UnimplementedJudgeServiceServer
	store store.JudgehostStoreInterface
	queue *queue.AsynqClient
}

// NewJudgeService creates a new judge service
func NewJudgeService(s store.JudgehostStoreInterface, asynqClient *queue.AsynqClient) *JudgeService {
	return &JudgeService{
		store: s,
		queue: asynqClient,
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
	pendingCount, _ := s.store.GetPendingTasksCount(ctx, fmt.Sprintf("judgehost-%s", req.JudgehostId))

	// If judgehost was idle and has capacity, potentially assign new tasks
	var assignedTasks []string
	if req.Status == pb.JudgehostStatus_JUDGEHOST_STATUS_IDLE && req.ActiveJobs < 5 {
		// This would be handled by the task assignment logic
		// For now, just acknowledge
	}

	return &pb.HeartbeatResponse{
		Acknowledged:     true,
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
		s.store.IncrementActiveJobs(ctx, judgehostID)
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
			Id:             j.ID,
			QueueName:      j.QueueName,
			Capabilities:   convertCapabilities(j),
			Status:         mapStringStatusToProto(j.Status),
			CurrentJobId:   j.CurrentJobID,
			LastPing:       j.LastPing.Format(time.RFC3339),
			ActiveJobs:     j.ActiveJobs,
			CompletedJobs:  j.CompletedJobs,
			RegisteredAt:   j.RegisteredAt.Format(time.RFC3339),
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
			Id:             j.ID,
			QueueName:      j.QueueName,
			Capabilities:   convertCapabilities(j),
			Status:         mapStringStatusToProto(j.Status),
			CurrentJobId:   j.CurrentJobID,
			LastPing:       j.LastPing.Format(time.RFC3339),
			ActiveJobs:     j.ActiveJobs,
			CompletedJobs:  j.CompletedJobs,
			RegisteredAt:   j.RegisteredAt.Format(time.RFC3339),
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

// Helper functions

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
		Languages:          j.Languages,
		MaxConcurrentJobs:  j.MaxConcurrent,
		MemoryLimit:        j.MemoryLimit,
		TimeLimit:          j.TimeLimit,
		SupportsInteractive: j.Interactive,
		SupportsSpecial:    j.Special,
		Extra:              j.Extra,
	}
}