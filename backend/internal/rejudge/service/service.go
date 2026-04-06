package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/rejudge/v1"
	conteststore "github.com/online-judge/backend/internal/contest/store"
	"github.com/online-judge/backend/internal/pkg/middleware"
	rejudgestore "github.com/online-judge/backend/internal/rejudge/store"
	submissionstore "github.com/online-judge/backend/internal/submission/store"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrRejudgeNotFound = errors.New("rejudge not found")
var ErrInvalidStatus = errors.New("invalid rejudge status for this operation")
var ErrRejudgeInProgress = errors.New("rejudge is still in progress")

type RejudgeService struct {
	pb.UnimplementedRejudgeServiceServer
	store          rejudgestore.RejudgeStoreInterface
	submissionStore submissionstore.SubmissionStoreInterface
	contestStore   conteststore.ContestStoreInterface
	redis          *redis.Client
}

func NewRejudgeService(
	s rejudgestore.RejudgeStoreInterface,
	subStore submissionstore.SubmissionStoreInterface,
	contestStore conteststore.ContestStoreInterface,
	redis *redis.Client,
) *RejudgeService {
	return &RejudgeService{
		store:          s,
		submissionStore: subStore,
		contestStore:   contestStore,
		redis:          redis,
	}
}

// CreateRejudge creates a new rejudging operation
func (s *RejudgeService) CreateRejudge(ctx context.Context, req *pb.CreateRejudgeRequest) (*pb.CreateRejudgeResponse, error) {
	// Get user ID from context (admin only)
	userID := middleware.GetUserID(ctx)
	if userID == "" {
		return nil, ErrUnauthorized
	}

	// Get submissions to rejudge
	submissionIDs, err := s.store.GetSubmissionsForRejudge(ctx, req.ContestId, req.ProblemId, req.FromVerdict, req.SubmissionIds)
	if err != nil {
		return nil, err
	}

	if len(submissionIDs) == 0 {
		return nil, fmt.Errorf("no submissions found matching criteria")
	}

	// Create rejudge record
	rejudgeID, affectedCount, err := s.store.CreateRejudge(ctx, userID, req.ContestId, req.ProblemId, req.FromVerdict, req.Reason, submissionIDs)
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
		err = s.store.CreateRejudgeSubmission(ctx, rejudgeID, submissionID, judging.ID, judging.Verdict)
		if err != nil {
			continue
		}

		// Get submission details
		sub, err := s.submissionStore.GetByID(ctx, submissionID)
		if err != nil {
			continue
		}

		// Queue submission for rejudging
		job := JudgeJob{
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
	s.store.UpdateRejudgeStatus(ctx, rejudgeID, "judging")

	return &pb.CreateRejudgeResponse{
		Id:           rejudgeID,
		AffectedCount: affectedCount,
		Status:        pb.RejudgeStatus_REJUDGE_STATUS_JUDGING,
	}, nil
}

// GetRejudge retrieves a rejudging operation
func (s *RejudgeService) GetRejudge(ctx context.Context, req *pb.GetRejudgeRequest) (*pb.GetRejudgeResponse, error) {
	rejudge, err := s.store.GetRejudge(ctx, req.Id)
	if err != nil {
		return nil, ErrRejudgeNotFound
	}

	// Get progress
	pending, judging, done, err := s.store.GetRejudgeProgress(ctx, req.Id)
	if err != nil {
		pending = 0
		judging = 0
		done = 0
	}

	return &pb.GetRejudgeResponse{
		Rejudge: &pb.Rejudge{
			Id:           rejudge.ID,
			UserId:       rejudge.UserID,
			ContestId:    rejudge.ContestID,
			ProblemId:    rejudge.ProblemID,
			Status:       mapStatusToProto(rejudge.Status),
			Reason:       rejudge.Reason,
			AffectedCount: rejudge.AffectedCount,
			JudgedCount:  done,
			PendingCount: pending + judging,
			CreatedAt:    timestamppb.New(rejudge.CreatedAt),
			StartedAt:    timestamppb.New(rejudge.StartedAt),
			FinishedAt:   timestamppb.New(rejudge.FinishedAt),
			AppliedAt:    timestamppb.New(rejudge.AppliedAt),
			RevertedAt:   timestamppb.New(rejudge.RevertedAt),
		},
	}, nil
}

// ListRejudges lists rejudging operations
func (s *RejudgeService) ListRejudges(ctx context.Context, req *pb.ListRejudgesRequest) (*pb.ListRejudgesResponse, error) {
	page := req.GetPagination().GetPage()
	pageSize := req.GetPagination().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	rejudges, total, err := s.store.ListRejudges(ctx, req.ContestId, req.ProblemId, mapStatusFromProto(req.Status), req.UserId, page, pageSize)
	if err != nil {
		return nil, err
	}

	var pbRejudges []*pb.Rejudge
	for _, r := range rejudges {
		pbRejudges = append(pbRejudges, &pb.Rejudge{
			Id:           r.ID,
			UserId:       r.UserID,
			ContestId:    r.ContestID,
			ProblemId:    r.ProblemID,
			Status:       mapStatusToProto(r.Status),
			Reason:       r.Reason,
			AffectedCount: r.AffectedCount,
			CreatedAt:    timestamppb.New(r.CreatedAt),
			StartedAt:    timestamppb.New(r.StartedAt),
			FinishedAt:   timestamppb.New(r.FinishedAt),
			AppliedAt:    timestamppb.New(r.AppliedAt),
			RevertedAt:   timestamppb.New(r.RevertedAt),
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
func (s *RejudgeService) CancelRejudge(ctx context.Context, req *pb.CancelRejudgeRequest) (*pb.CancelRejudgeResponse, error) {
	rejudge, err := s.store.GetRejudge(ctx, req.Id)
	if err != nil {
		return nil, ErrRejudgeNotFound
	}

	// Can only cancel pending or judging rejudges
	if rejudge.Status != "pending" && rejudge.Status != "judging" {
		return nil, ErrInvalidStatus
	}

	err = s.store.CancelRejudge(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.CancelRejudgeResponse{
		Id:     req.Id,
		Status: pb.RejudgeStatus_REJUDGE_STATUS_CANCELLED,
	}, nil
}

// ApplyRejudge applies the rejudge results
func (s *RejudgeService) ApplyRejudge(ctx context.Context, req *pb.ApplyRejudgeRequest) (*pb.ApplyRejudgeResponse, error) {
	rejudge, err := s.store.GetRejudge(ctx, req.Id)
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

	verdictChanges, err := s.store.ApplyRejudge(ctx, req.Id)
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
		Id:            req.Id,
		VerdictsChanged: verdictChanges,
		Status:         pb.RejudgeStatus_REJUDGE_STATUS_APPLIED,
	}, nil
}

// RevertRejudge reverts the rejudge results
func (s *RejudgeService) RevertRejudge(ctx context.Context, req *pb.RevertRejudgeRequest) (*pb.RevertRejudgeResponse, error) {
	rejudge, err := s.store.GetRejudge(ctx, req.Id)
	if err != nil {
		return nil, ErrRejudgeNotFound
	}

	// Can only revert applied rejudges
	if rejudge.Status != "applied" {
		return nil, ErrInvalidStatus
	}

	restoredCount, err := s.store.RevertRejudge(ctx, req.Id)
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
		Id:             req.Id,
		VerdictsRestored: restoredCount,
		Status:          pb.RejudgeStatus_REJUDGE_STATUS_REVERTED,
	}, nil
}

// GetRejudgeSubmissions retrieves submissions for a rejudging operation
func (s *RejudgeService) GetRejudgeSubmissions(ctx context.Context, req *pb.GetRejudgeSubmissionsRequest) (*pb.GetRejudgeSubmissionsResponse, error) {
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
		status = mapSubmissionStatusFromProto(req.Status)
	}

	submissions, total, err := s.store.GetRejudgeSubmissions(ctx, req.RejudgeId, req.OnlyChanged, status, page, pageSize)
	if err != nil {
		return nil, err
	}

	var pbSubmissions []*pb.RejudgeSubmission
	for _, rs := range submissions {
		pbSubmissions = append(pbSubmissions, &pb.RejudgeSubmission{
			SubmissionId:     rs.SubmissionID,
			OriginalJudgingId: rs.OriginalJudgingID,
			NewJudgingId:     rs.NewJudgingID,
			OriginalVerdict:  rs.OriginalVerdict,
			NewVerdict:       rs.NewVerdict,
			VerdictChanged:   rs.VerdictChanged,
			Status:           mapSubmissionStatusToProto(rs.Status),
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
func (s *RejudgeService) InternalUpdateRejudgeSubmission(ctx context.Context, req *pb.InternalUpdateRejudgeSubmissionRequest) (*pb.InternalUpdateRejudgeSubmissionResponse, error) {
	err := s.store.UpdateRejudgeSubmission(ctx, req.RejudgeId, req.SubmissionId, req.NewJudgingId, req.NewVerdict)
	if err != nil {
		return nil, err
	}

	// Check if all submissions are done
	pending, judging, _, err := s.store.GetRejudgeProgress(ctx, req.RejudgeId)
	if err == nil && pending == 0 && judging == 0 {
		// All done, update rejudge status
		s.store.UpdateRejudgeStatus(ctx, req.RejudgeId, "judged")
	}

	return &pb.InternalUpdateRejudgeSubmissionResponse{
		Status: "updated",
	}, nil
}

// Helper functions
func mapStatusToProto(status string) pb.RejudgeStatus {
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

func mapStatusFromProto(status pb.RejudgeStatus) string {
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

func mapSubmissionStatusToProto(status string) pb.RejudgeSubmissionStatus {
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

func mapSubmissionStatusFromProto(status pb.RejudgeSubmissionStatus) string {
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

func (s *RejudgeService) pushToQueue(ctx context.Context, job *JudgeJob) error {
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

// JudgeJob extended with rejudge_id
type JudgeJob struct {
	SubmissionID string    `json:"submission_id"`
	ProblemID    string    `json:"problem_id"`
	Language     string    `json:"language"`
	Priority     int       `json:"priority"`
	SubmitTime   time.Time `json:"submit_time"`
	RejudgeID    string    `json:"rejudge_id,omitempty"`
}