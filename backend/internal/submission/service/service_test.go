package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/online-judge/backend/gen/go/common/v1"
	pb "github.com/online-judge/backend/gen/go/submission/v1"
	"github.com/online-judge/backend/internal/submission/store"
)

func TestSubmissionService_CreateSubmission(t *testing.T) {
	// Note: CreateSubmission requires Redis for queue operations.
	// These tests focus on store operations. Full integration tests
	// should use miniredis or a real Redis instance.
	t.Skip("CreateSubmission requires Redis for queue operations - use integration tests")
}

func TestSubmissionService_GetSubmission(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockSubmissionStore)
		request *pb.GetSubmissionRequest
		want    func(t *testing.T, resp *pb.GetSubmissionResponse, err error)
	}{
		{
			name: "get existing submission",
			setup: func(m *store.MockSubmissionStore) {
				m.Submissions["sub-1"] = &store.Submission{
					ID:         "sub-1",
					UserID:     "user-1",
					ProblemID:  "prob-1",
					LanguageID: "cpp",
					SourceCode: "int main() { return 0; }",
				}
				m.Judgings["jud-1"] = &store.Judging{
					ID:           "jud-1",
					SubmissionID: "sub-1",
					JudgehostID:  "host-1",
					Verdict:      "correct",
					MaxRuntime:   0.5,
					MaxMemory:    1024,
					Valid:        true,
				}
			},
			request: &pb.GetSubmissionRequest{Id: "sub-1"},
			want: func(t *testing.T, resp *pb.GetSubmissionResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "sub-1", resp.Submission.Id)
				assert.Equal(t, "prob-1", resp.Submission.ProblemId)
				assert.NotNil(t, resp.LatestJudging)
				assert.Equal(t, pb.Verdict_VERDICT_CORRECT, resp.LatestJudging.Verdict)
			},
		},
		{
			name:    "get non-existent submission",
			setup:   func(m *store.MockSubmissionStore) {},
			request: &pb.GetSubmissionRequest{Id: "non-existent"},
			want: func(t *testing.T, resp *pb.GetSubmissionResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockSubmissionStore) {
				m.GetError = assert.AnError
			},
			request: &pb.GetSubmissionRequest{Id: "sub-1"},
			want: func(t *testing.T, resp *pb.GetSubmissionResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockSubmissionStore()
			tt.setup(mockStore)

			service := NewSubmissionService(mockStore, nil, nil, nil, false, false)
			resp, err := service.GetSubmission(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestSubmissionService_ListSubmissions(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockSubmissionStore)
		request *pb.ListSubmissionsRequest
		want    func(t *testing.T, resp *pb.ListSubmissionsResponse, err error)
	}{
		{
			name: "list all submissions",
			setup: func(m *store.MockSubmissionStore) {
				m.Submissions["sub-1"] = &store.Submission{
					ID:         "sub-1",
					UserID:     "user-1",
					ProblemID:  "prob-1",
					LanguageID: "cpp",
				}
				m.Submissions["sub-2"] = &store.Submission{
					ID:         "sub-2",
					UserID:     "user-2",
					ProblemID:  "prob-2",
					LanguageID: "python",
				}
			},
			request: &pb.ListSubmissionsRequest{
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 2)
				assert.Equal(t, int32(2), resp.Pagination.Total)
			},
		},
		{
			name: "list submissions by user",
			setup: func(m *store.MockSubmissionStore) {
				m.Submissions["sub-1"] = &store.Submission{
					ID:        "sub-1",
					UserID:    "user-1",
					ProblemID: "prob-1",
				}
				m.Submissions["sub-2"] = &store.Submission{
					ID:        "sub-2",
					UserID:    "user-2",
					ProblemID: "prob-2",
				}
			},
			request: &pb.ListSubmissionsRequest{
				UserId: "user-1",
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 1)
				assert.Equal(t, "sub-1", resp.Submissions[0].Id)
			},
		},
		{
			name: "list submissions by problem",
			setup: func(m *store.MockSubmissionStore) {
				m.Submissions["sub-1"] = &store.Submission{
					ID:        "sub-1",
					UserID:    "user-1",
					ProblemID: "prob-1",
				}
				m.Submissions["sub-2"] = &store.Submission{
					ID:        "sub-2",
					UserID:    "user-2",
					ProblemID: "prob-2",
				}
			},
			request: &pb.ListSubmissionsRequest{
				ProblemId: "prob-1",
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListSubmissionsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Submissions, 1)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockSubmissionStore) {
				m.ListError = assert.AnError
			},
			request: &pb.ListSubmissionsRequest{
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListSubmissionsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockSubmissionStore()
			tt.setup(mockStore)

			service := NewSubmissionService(mockStore, nil, nil, nil, false, false)
			resp, err := service.ListSubmissions(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestSubmissionService_GetJudging(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockSubmissionStore)
		request *pb.GetJudgingRequest
		want    func(t *testing.T, resp *pb.GetJudgingResponse, err error)
	}{
		{
			name: "get judging by submission id",
			setup: func(m *store.MockSubmissionStore) {
				m.Judgings["jud-1"] = &store.Judging{
					ID:           "jud-1",
					SubmissionID: "sub-1",
					JudgehostID:  "host-1",
					Verdict:      "correct",
					MaxRuntime:   0.5,
					MaxMemory:    1024,
					Valid:        true,
				}
			},
			request: &pb.GetJudgingRequest{SubmissionId: "sub-1"},
			want: func(t *testing.T, resp *pb.GetJudgingResponse, err error) {
				require.NoError(t, err)
				assert.NotNil(t, resp.Judging)
				assert.Equal(t, pb.Verdict_VERDICT_CORRECT, resp.Judging.Verdict)
			},
		},
		{
			name: "get judging by judging id",
			setup: func(m *store.MockSubmissionStore) {
				m.Judgings["jud-1"] = &store.Judging{
					ID:           "jud-1",
					SubmissionID: "sub-1",
					JudgehostID:  "host-1",
					Verdict:      "wrong-answer",
					MaxRuntime:   1.0,
					MaxMemory:    2048,
					Valid:        true,
				}
			},
			request: &pb.GetJudgingRequest{JudgingId: "jud-1"},
			want: func(t *testing.T, resp *pb.GetJudgingResponse, err error) {
				require.NoError(t, err)
				assert.NotNil(t, resp.Judging)
				assert.Equal(t, pb.Verdict_VERDICT_WRONG_ANSWER, resp.Judging.Verdict)
			},
		},
		{
			name:    "judging not found",
			setup:   func(m *store.MockSubmissionStore) {},
			request: &pb.GetJudgingRequest{SubmissionId: "sub-nonexistent"},
			want: func(t *testing.T, resp *pb.GetJudgingResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockSubmissionStore()
			tt.setup(mockStore)

			service := NewSubmissionService(mockStore, nil, nil, nil, false, false)
			resp, err := service.GetJudging(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestSubmissionService_GetJudgingRuns(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockSubmissionStore)
		request *pb.GetJudgingRunsRequest
		want    func(t *testing.T, resp *pb.GetJudgingRunsResponse, err error)
	}{
		{
			name: "get judging runs",
			setup: func(m *store.MockSubmissionStore) {
				m.Judgings["jud-1"] = &store.Judging{
					ID:           "jud-1",
					SubmissionID: "sub-1",
					Valid:        true,
				}
				m.JudgingRuns["jud-1"] = []*store.JudgingRun{
					{
						ID:         "run-1",
						JudgingID:  "jud-1",
						TestCaseID: "tc-1",
						Rank:       1,
						Verdict:    "correct",
						Runtime:    0.1,
						Memory:     512,
					},
					{
						ID:         "run-2",
						JudgingID:  "jud-1",
						TestCaseID: "tc-2",
						Rank:       2,
						Verdict:    "correct",
						Runtime:    0.2,
						Memory:     768,
					},
				}
			},
			request: &pb.GetJudgingRunsRequest{SubmissionId: "sub-1"},
			want: func(t *testing.T, resp *pb.GetJudgingRunsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Runs, 2)
				assert.Equal(t, pb.Verdict_VERDICT_CORRECT, resp.Runs[0].Verdict)
			},
		},
		{
			name:    "no runs for submission",
			setup:   func(m *store.MockSubmissionStore) {},
			request: &pb.GetJudgingRunsRequest{SubmissionId: "sub-nonexistent"},
			want: func(t *testing.T, resp *pb.GetJudgingRunsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockSubmissionStore()
			tt.setup(mockStore)

			service := NewSubmissionService(mockStore, nil, nil, nil, false, false)
			resp, err := service.GetJudgingRuns(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestSubmissionService_InternalCreateJudging(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockSubmissionStore)
		request *pb.InternalCreateJudgingRequest
		want    func(t *testing.T, resp *pb.InternalCreateJudgingResponse, err error)
	}{
		{
			name:  "create judging successfully",
			setup: func(m *store.MockSubmissionStore) {},
			request: &pb.InternalCreateJudgingRequest{
				SubmissionId: "sub-1",
				JudgehostId:  "host-1",
			},
			want: func(t *testing.T, resp *pb.InternalCreateJudgingResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.JudgingId)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockSubmissionStore) {
				m.JudgingError = assert.AnError
			},
			request: &pb.InternalCreateJudgingRequest{
				SubmissionId: "sub-1",
				JudgehostId:  "host-1",
			},
			want: func(t *testing.T, resp *pb.InternalCreateJudgingResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockSubmissionStore()
			tt.setup(mockStore)

			service := NewSubmissionService(mockStore, nil, nil, nil, false, false)
			resp, err := service.InternalCreateJudging(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestSubmissionService_InternalUpdateJudging(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockSubmissionStore)
		request *pb.InternalUpdateJudgingRequest
		want    func(t *testing.T, resp *pb.InternalUpdateJudgingResponse, err error)
	}{
		{
			name: "update judging with correct verdict",
			setup: func(m *store.MockSubmissionStore) {
				m.Judgings["jud-1"] = &store.Judging{
					ID:           "jud-1",
					SubmissionID: "sub-1",
				}
			},
			request: &pb.InternalUpdateJudgingRequest{
				JudgingId:  "jud-1",
				Verdict:    "correct",
				MaxRuntime: 0.5,
				MaxMemory:  1024,
			},
			want: func(t *testing.T, resp *pb.InternalUpdateJudgingResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "jud-1", resp.JudgingId)
				assert.Equal(t, "updated", resp.Status)
			},
		},
		{
			name: "update judging with compiler error",
			setup: func(m *store.MockSubmissionStore) {
				m.Judgings["jud-1"] = &store.Judging{
					ID:           "jud-1",
					SubmissionID: "sub-1",
				}
			},
			request: &pb.InternalUpdateJudgingRequest{
				JudgingId:    "jud-1",
				Verdict:      "compiler-error",
				MaxRuntime:   0,
				MaxMemory:    0,
				CompileError: "undefined variable",
			},
			want: func(t *testing.T, resp *pb.InternalUpdateJudgingResponse, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:    "judging not found",
			setup:   func(m *store.MockSubmissionStore) {},
			request: &pb.InternalUpdateJudgingRequest{JudgingId: "nonexistent"},
			want: func(t *testing.T, resp *pb.InternalUpdateJudgingResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockSubmissionStore()
			tt.setup(mockStore)

			service := NewSubmissionService(mockStore, nil, nil, nil, false, false)
			resp, err := service.InternalUpdateJudging(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestSubmissionService_InternalCreateJudgingRun(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockSubmissionStore)
		request *pb.InternalCreateJudgingRunRequest
		want    func(t *testing.T, resp *pb.InternalCreateJudgingRunResponse, err error)
	}{
		{
			name: "create judging run successfully",
			setup: func(m *store.MockSubmissionStore) {
				m.Judgings["jud-1"] = &store.Judging{ID: "jud-1"}
			},
			request: &pb.InternalCreateJudgingRunRequest{
				JudgingId:  "jud-1",
				TestCaseId: "tc-1",
				Rank:       1,
				Verdict:    "correct",
				Runtime:    0.1,
				WallTime:   0.15,
				Memory:     512,
			},
			want: func(t *testing.T, resp *pb.InternalCreateJudgingRunResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "created", resp.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockSubmissionStore()
			tt.setup(mockStore)

			service := NewSubmissionService(mockStore, nil, nil, nil, false, false)
			resp, err := service.InternalCreateJudgingRun(context.Background(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestMapVerdictToProto(t *testing.T) {
	tests := []struct {
		input    string
		expected pb.Verdict
	}{
		{"correct", pb.Verdict_VERDICT_CORRECT},
		{"wrong-answer", pb.Verdict_VERDICT_WRONG_ANSWER},
		{"timelimit", pb.Verdict_VERDICT_TIMELIMIT},
		{"time-limit", pb.Verdict_VERDICT_TIMELIMIT},
		{"memory-limit", pb.Verdict_VERDICT_MEMORY_LIMIT},
		{"run-error", pb.Verdict_VERDICT_RUN_ERROR},
		{"compiler-error", pb.Verdict_VERDICT_COMPILER_ERROR},
		{"output-limit", pb.Verdict_VERDICT_OUTPUT_LIMIT},
		{"presentation", pb.Verdict_VERDICT_PRESENTATION},
		{"unknown", pb.Verdict_VERDICT_UNSPECIFIED},
		{"", pb.Verdict_VERDICT_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapVerdictToProto(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
