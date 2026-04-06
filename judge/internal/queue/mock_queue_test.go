package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockJudgeQueue_Push(t *testing.T) {
	tests := []struct {
		name    string
		job     *JudgeJob
		setup   func(m *MockJudgeQueue)
		wantErr bool
	}{
		{
			name: "push job successfully",
			job: &JudgeJob{
				SubmissionID: "sub-1",
				ProblemID:    "prob-1",
				Language:     "cpp",
				Priority:     0,
				SubmitTime:   time.Now(),
			},
			setup:   func(m *MockJudgeQueue) {},
			wantErr: false,
		},
		{
			name: "push job with error",
			job: &JudgeJob{
				SubmissionID: "sub-2",
				ProblemID:    "prob-1",
				Language:     "python",
			},
			setup: func(m *MockJudgeQueue) {
				m.PushError = assert.AnError
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			err := mockQueue.Push(context.Background(), tt.job)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, mockQueue.Jobs, 1)
				assert.Equal(t, tt.job.SubmissionID, mockQueue.Jobs[0].SubmissionID)
			}
		})
	}
}

func TestMockJudgeQueue_Pop(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockJudgeQueue)
		wantJob *JudgeJob
		wantErr bool
	}{
		{
			name: "pop job from queue",
			setup: func(m *MockJudgeQueue) {
				m.Jobs = append(m.Jobs, &JudgeJob{
					SubmissionID: "sub-1",
					ProblemID:    "prob-1",
					Language:     "cpp",
				})
			},
			wantJob: &JudgeJob{SubmissionID: "sub-1"},
			wantErr: false,
		},
		{
			name:    "pop from empty queue",
			setup:   func(m *MockJudgeQueue) {},
			wantJob: nil,
			wantErr: false,
		},
		{
			name: "pop error",
			setup: func(m *MockJudgeQueue) {
				m.PopError = assert.AnError
			},
			wantJob: nil,
			wantErr: true,
		},
		{
			name: "pop from pre-configured jobs",
			setup: func(m *MockJudgeQueue) {
				m.PopJobs = []*JudgeJob{
					{SubmissionID: "sub-1"},
					{SubmissionID: "sub-2"},
				}
			},
			wantJob: &JudgeJob{SubmissionID: "sub-1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			job, err := mockQueue.Pop(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.wantJob != nil {
					assert.Equal(t, tt.wantJob.SubmissionID, job.SubmissionID)
				} else {
					assert.Nil(t, job)
				}
			}
		})
	}
}

func TestMockJudgeQueue_PushResult(t *testing.T) {
	tests := []struct {
		name    string
		result  *JudgeResult
		setup   func(m *MockJudgeQueue)
		wantErr bool
	}{
		{
			name: "push result successfully",
			result: &JudgeResult{
				SubmissionID: "sub-1",
				Verdict:      "correct",
				Runtime:      0.5,
				Memory:       1024,
			},
			setup:   func(m *MockJudgeQueue) {},
			wantErr: false,
		},
		{
			name: "push result error",
			result: &JudgeResult{
				SubmissionID: "sub-2",
				Verdict:      "wrong-answer",
			},
			setup: func(m *MockJudgeQueue) {
				m.PushResultError = assert.AnError
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			err := mockQueue.PushResult(context.Background(), tt.result)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, mockQueue.Results, 1)
			}
		})
	}
}

func TestMockJudgeQueue_FetchSubmission(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockJudgeQueue)
		want    *SubmissionDetails
		wantErr bool
	}{
		{
			name: "fetch submission successfully",
			setup: func(m *MockJudgeQueue) {
				m.SubmissionDetails = &SubmissionDetails{
					ID:         "sub-1",
					UserID:     "user-1",
					ProblemID:  "prob-1",
					LanguageID: "cpp",
					SourceCode: "int main() { return 0; }",
				}
			},
			want: &SubmissionDetails{
				ID:         "sub-1",
				UserID:     "user-1",
				ProblemID:  "prob-1",
				LanguageID: "cpp",
				SourceCode: "int main() { return 0; }",
			},
			wantErr: false,
		},
		{
			name: "fetch submission default",
			setup: func(m *MockJudgeQueue) {
				// No custom submission details
			},
			want: &SubmissionDetails{
				ID:         "sub-1",
				LanguageID: "cpp",
			},
			wantErr: false,
		},
		{
			name: "fetch submission error",
			setup: func(m *MockJudgeQueue) {
				m.FetchSubmissionError = assert.AnError
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			result, err := mockQueue.FetchSubmission(context.Background(), "sub-1")

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.ID, result.ID)
			}
		})
	}
}

func TestMockJudgeQueue_FetchProblem(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockJudgeQueue)
		want    *ProblemDetails
		wantErr bool
	}{
		{
			name: "fetch problem successfully",
			setup: func(m *MockJudgeQueue) {
				m.ProblemDetails = &ProblemDetails{
					ID:           "prob-1",
					TimeLimit:    2.0,
					MemoryLimit:  512000,
					OutputLimit:  10240,
					ProcessLimit: 1,
				}
			},
			want: &ProblemDetails{
				ID:           "prob-1",
				TimeLimit:    2.0,
				MemoryLimit:  512000,
				OutputLimit:  10240,
				ProcessLimit: 1,
			},
			wantErr: false,
		},
		{
			name: "fetch problem error",
			setup: func(m *MockJudgeQueue) {
				m.FetchProblemError = assert.AnError
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			result, err := mockQueue.FetchProblem(context.Background(), "prob-1")

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.ID, result.ID)
			}
		})
	}
}

func TestMockJudgeQueue_FetchTestCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockJudgeQueue)
		wantLen int
		wantErr bool
	}{
		{
			name: "fetch test cases successfully",
			setup: func(m *MockJudgeQueue) {
				m.TestCasesList = []*TestCase{
					{ID: "tc-1", ProblemID: "prob-1", Rank: 1, IsSample: true},
					{ID: "tc-2", ProblemID: "prob-1", Rank: 2, IsSample: false},
				}
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "fetch default test cases",
			setup:   func(m *MockJudgeQueue) {},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "fetch test cases error",
			setup: func(m *MockJudgeQueue) {
				m.FetchTestCasesError = assert.AnError
			},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			result, err := mockQueue.FetchTestCases(context.Background(), "prob-1")

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.wantLen)
			}
		})
	}
}

func TestMockJudgeQueue_FetchTestCaseData(t *testing.T) {
	tests := []struct {
		name     string
		testCase *TestCase
		setup    func(m *MockJudgeQueue)
		wantErr  bool
	}{
		{
			name: "fetch test case data from map",
			testCase: &TestCase{ID: "tc-1"},
			setup: func(m *MockJudgeQueue) {
				m.TestCaseDataMap["tc-1"] = &TestCaseData{
					Input:  []byte("1 2\n"),
					Output: []byte("3\n"),
				}
			},
			wantErr: false,
		},
		{
			name:     "fetch default test case data",
			testCase: &TestCase{ID: "tc-2"},
			setup:    func(m *MockJudgeQueue) {},
			wantErr:  false,
		},
		{
			name:     "fetch test case data error",
			testCase: &TestCase{ID: "tc-3"},
			setup: func(m *MockJudgeQueue) {
				m.FetchTestCaseDataError = assert.AnError
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			result, err := mockQueue.FetchTestCaseData(context.Background(), tt.testCase)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestMockJudgeQueue_CreateJudging(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockJudgeQueue)
		wantID  string
		wantErr bool
	}{
		{
			name: "create judging successfully",
			setup: func(m *MockJudgeQueue) {
				m.CreatedJudgingID = "jud-123"
			},
			wantID:  "jud-123",
			wantErr: false,
		},
		{
			name:    "create judging with default ID",
			setup:   func(m *MockJudgeQueue) {},
			wantID:  "judging-sub-1",
			wantErr: false,
		},
		{
			name: "create judging error",
			setup: func(m *MockJudgeQueue) {
				m.CreateJudgingError = assert.AnError
			},
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			id, err := mockQueue.CreateJudging(context.Background(), "sub-1", "host-1")

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestMockJudgeQueue_UpdateJudging(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockJudgeQueue)
		wantErr bool
	}{
		{
			name:    "update judging successfully",
			setup:   func(m *MockJudgeQueue) {},
			wantErr: false,
		},
		{
			name: "update judging error",
			setup: func(m *MockJudgeQueue) {
				m.UpdateJudgingError = assert.AnError
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			err := mockQueue.UpdateJudging(context.Background(), "jud-1", &JudgeResult{
				Verdict: "correct",
			})

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMockJudgeQueue_PushJudgingResult(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(m *MockJudgeQueue)
		wantErr bool
	}{
		{
			name:    "push judging result successfully",
			setup:   func(m *MockJudgeQueue) {},
			wantErr: false,
		},
		{
			name: "push judging result error",
			setup: func(m *MockJudgeQueue) {
				m.PushJudgingResultError = assert.AnError
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockQueue := NewMockJudgeQueue()
			tt.setup(mockQueue)

			err := mockQueue.PushJudgingResult(
				context.Background(),
				&JudgeResult{Verdict: "correct"},
				"jud-1",
				[]*TestCaseResult{},
			)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMockJudgeQueue_Length(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	mockQueue.Jobs = append(mockQueue.Jobs, &JudgeJob{SubmissionID: "sub-1"})
	mockQueue.Jobs = append(mockQueue.Jobs, &JudgeJob{SubmissionID: "sub-2"})

	length, err := mockQueue.Length(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(2), length)
}

func TestMockJudgeQueue_Ping(t *testing.T) {
	mockQueue := NewMockJudgeQueue()
	err := mockQueue.Ping(context.Background())
	require.NoError(t, err)
}