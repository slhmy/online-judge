package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/slhmy/online-judge/backend/internal/problem/store"
	commonv1 "github.com/slhmy/online-judge/gen/go/common/v1"
	pb "github.com/slhmy/online-judge/gen/go/problem/v1"
)

func testAdminContext() context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-role", "admin"))
}

func TestProblemService_ListProblems(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockProblemStore)
		request *pb.ListProblemsRequest
		want    func(t *testing.T, resp *pb.ListProblemsResponse, err error)
	}{
		{
			name: "list all published problems",
			setup: func(m *store.MockProblemStore) {
				m.Problems["prob-1"] = &pb.Problem{
					Id:          "prob-1",
					ExternalId:  "A",
					Name:        "Problem A",
					Difficulty:  "easy",
					TimeLimit:   1.0,
					MemoryLimit: 256,
					Points:      100,
					IsPublished: true,
					AllowSubmit: true,
				}
				m.Problems["prob-2"] = &pb.Problem{
					Id:          "prob-2",
					ExternalId:  "B",
					Name:        "Problem B",
					Difficulty:  "medium",
					TimeLimit:   2.0,
					MemoryLimit: 512,
					Points:      200,
					IsPublished: true,
					AllowSubmit: true,
				}
				m.Problems["prob-3"] = &pb.Problem{
					Id:          "prob-3",
					ExternalId:  "C",
					Name:        "Problem C",
					Difficulty:  "hard",
					TimeLimit:   3.0,
					MemoryLimit: 1024,
					Points:      300,
					IsPublished: false,
					AllowSubmit: false,
				}
			},
			request: &pb.ListProblemsRequest{
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListProblemsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Problems, 2) // Only published
				assert.Equal(t, int32(2), resp.Pagination.Total)
			},
		},
		{
			name: "list problems with pagination",
			setup: func(m *store.MockProblemStore) {
				for i := 1; i <= 25; i++ {
					m.Problems[string(rune('0'+i))] = &pb.Problem{
						Id:          string(rune('0' + i)),
						Name:        "Problem",
						IsPublished: true,
						AllowSubmit: true,
					}
				}
			},
			request: &pb.ListProblemsRequest{
				Pagination: &commonv1.Pagination{
					Page:     2,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListProblemsResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Problems, 10)
				assert.Equal(t, int32(25), resp.Pagination.Total)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockProblemStore) {
				m.ListError = assert.AnError
			},
			request: &pb.ListProblemsRequest{
				Pagination: &commonv1.Pagination{
					Page:     1,
					PageSize: 10,
				},
			},
			want: func(t *testing.T, resp *pb.ListProblemsResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil) // nil redis for tests
			resp, err := service.ListProblems(testAdminContext(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestProblemService_GetProblem(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockProblemStore)
		request *pb.GetProblemRequest
		want    func(t *testing.T, resp *pb.GetProblemResponse, err error)
	}{
		{
			name: "get existing problem",
			setup: func(m *store.MockProblemStore) {
				m.Problems["prob-1"] = &pb.Problem{
					Id:          "prob-1",
					ExternalId:  "A",
					Name:        "Problem A",
					Difficulty:  "easy",
					TimeLimit:   1.0,
					MemoryLimit: 256,
					Points:      100,
					IsPublished: true,
				}
				m.TestCases["prob-1"] = []*pb.TestCase{
					{
						Id:        "tc-1",
						ProblemId: "prob-1",
						Rank:      1,
						IsSample:  true,
					},
				}
			},
			request: &pb.GetProblemRequest{Id: "prob-1"},
			want: func(t *testing.T, resp *pb.GetProblemResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "prob-1", resp.Problem.Id)
				assert.Equal(t, "Problem A", resp.Problem.Name)
				assert.Len(t, resp.SampleTestCases, 1)
			},
		},
		{
			name:    "get non-existent problem",
			setup:   func(m *store.MockProblemStore) {},
			request: &pb.GetProblemRequest{Id: "non-existent"},
			want: func(t *testing.T, resp *pb.GetProblemResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockProblemStore) {
				m.GetError = assert.AnError
			},
			request: &pb.GetProblemRequest{Id: "prob-1"},
			want: func(t *testing.T, resp *pb.GetProblemResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil)
			resp, err := service.GetProblem(testAdminContext(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestProblemService_CreateProblem(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockProblemStore)
		request *pb.CreateProblemRequest
		want    func(t *testing.T, resp *pb.CreateProblemResponse, err error)
	}{
		{
			name:  "create problem successfully",
			setup: func(m *store.MockProblemStore) {},
			request: &pb.CreateProblemRequest{
				ExternalId:  "A",
				Name:        "New Problem",
				TimeLimit:   1.5,
				MemoryLimit: 512,
				OutputLimit: 1024,
				Difficulty:  "medium",
				Points:      150,
			},
			want: func(t *testing.T, resp *pb.CreateProblemResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Id)
			},
		},
		{
			name: "store error on create",
			setup: func(m *store.MockProblemStore) {
				m.CreateError = assert.AnError
			},
			request: &pb.CreateProblemRequest{
				Name: "Problem",
			},
			want: func(t *testing.T, resp *pb.CreateProblemResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil)
			resp, err := service.CreateProblem(testAdminContext(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestProblemService_UpdateProblem(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockProblemStore)
		request *pb.UpdateProblemRequest
		want    func(t *testing.T, resp *pb.UpdateProblemResponse, err error)
	}{
		{
			name: "update existing problem",
			setup: func(m *store.MockProblemStore) {
				m.Problems["prob-1"] = &pb.Problem{
					Id:          "prob-1",
					Name:        "Old Name",
					TimeLimit:   1.0,
					MemoryLimit: 256,
					IsPublished: true,
					AllowSubmit: true,
				}
			},
			request: &pb.UpdateProblemRequest{
				Id:          "prob-1",
				Name:        "New Name",
				TimeLimit:   2.0,
				MemoryLimit: 512,
				IsPublished: true,
				AllowSubmit: true,
			},
			want: func(t *testing.T, resp *pb.UpdateProblemResponse, err error) {
				require.NoError(t, err)
				assert.Equal(t, "New Name", resp.Problem.Name)
				assert.Equal(t, 2.0, resp.Problem.TimeLimit)
			},
		},
		{
			name:    "update non-existent problem",
			setup:   func(m *store.MockProblemStore) {},
			request: &pb.UpdateProblemRequest{Id: "non-existent"},
			want: func(t *testing.T, resp *pb.UpdateProblemResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockProblemStore) {
				m.Problems["prob-1"] = &pb.Problem{Id: "prob-1"}
				m.UpdateError = assert.AnError
			},
			request: &pb.UpdateProblemRequest{Id: "prob-1"},
			want: func(t *testing.T, resp *pb.UpdateProblemResponse, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil)
			resp, err := service.UpdateProblem(testAdminContext(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestProblemService_DeleteProblem(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockProblemStore)
		request *pb.DeleteProblemRequest
		want    func(t *testing.T, resp *emptypb.Empty, err error)
	}{
		{
			name: "delete existing problem",
			setup: func(m *store.MockProblemStore) {
				m.Problems["prob-1"] = &pb.Problem{Id: "prob-1"}
			},
			request: &pb.DeleteProblemRequest{Id: "prob-1"},
			want: func(t *testing.T, resp *emptypb.Empty, err error) {
				require.NoError(t, err)
				assert.NotNil(t, resp)
			},
		},
		{
			name:    "delete non-existent problem",
			setup:   func(m *store.MockProblemStore) {},
			request: &pb.DeleteProblemRequest{Id: "non-existent"},
			want: func(t *testing.T, resp *emptypb.Empty, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
		{
			name: "store error",
			setup: func(m *store.MockProblemStore) {
				m.DeleteError = assert.AnError
			},
			request: &pb.DeleteProblemRequest{Id: "prob-1"},
			want: func(t *testing.T, resp *emptypb.Empty, err error) {
				require.Error(t, err)
				assert.Nil(t, resp)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil)
			resp, err := service.DeleteProblem(testAdminContext(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestProblemService_ListTestCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockProblemStore)
		request *pb.ListTestCasesRequest
		want    func(t *testing.T, resp *pb.ListTestCasesResponse, err error)
	}{
		{
			name: "list all test cases",
			setup: func(m *store.MockProblemStore) {
				m.TestCases["prob-1"] = []*pb.TestCase{
					{Id: "tc-1", ProblemId: "prob-1", Rank: 1, IsSample: true},
					{Id: "tc-2", ProblemId: "prob-1", Rank: 2, IsSample: false},
					{Id: "tc-3", ProblemId: "prob-1", Rank: 3, IsSample: false},
				}
			},
			request: &pb.ListTestCasesRequest{
				ProblemId:   "prob-1",
				SamplesOnly: false,
			},
			want: func(t *testing.T, resp *pb.ListTestCasesResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.TestCases, 3)
			},
		},
		{
			name: "list only sample test cases",
			setup: func(m *store.MockProblemStore) {
				m.TestCases["prob-1"] = []*pb.TestCase{
					{Id: "tc-1", ProblemId: "prob-1", Rank: 1, IsSample: true},
					{Id: "tc-2", ProblemId: "prob-1", Rank: 2, IsSample: false},
				}
			},
			request: &pb.ListTestCasesRequest{
				ProblemId:   "prob-1",
				SamplesOnly: true,
			},
			want: func(t *testing.T, resp *pb.ListTestCasesResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.TestCases, 1)
				assert.True(t, resp.TestCases[0].IsSample)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil)
			resp, err := service.ListTestCases(testAdminContext(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestProblemService_CreateTestCase(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*store.MockProblemStore)
		request *pb.CreateTestCaseRequest
		want    func(t *testing.T, resp *pb.CreateTestCaseResponse, err error)
	}{
		{
			name:  "create test case successfully",
			setup: func(m *store.MockProblemStore) {},
			request: &pb.CreateTestCaseRequest{
				ProblemId:   "prob-1",
				Rank:        1,
				IsSample:    true,
				Description: "Sample test case",
			},
			want: func(t *testing.T, resp *pb.CreateTestCaseResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Id)
				assert.NotEmpty(t, resp.InputPath)
				assert.NotEmpty(t, resp.OutputPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil)
			resp, err := service.CreateTestCase(testAdminContext(), tt.request)

			tt.want(t, resp, err)
		})
	}
}

func TestProblemService_ListLanguages(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*store.MockProblemStore)
		want  func(t *testing.T, resp *pb.ListLanguagesResponse, err error)
	}{
		{
			name:  "list default languages",
			setup: func(m *store.MockProblemStore) {},
			want: func(t *testing.T, resp *pb.ListLanguagesResponse, err error) {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Languages)
				// Verify all required languages are present
				langNames := make(map[string]bool)
				for _, lang := range resp.Languages {
					langNames[lang.Name] = true
					assert.True(t, lang.AllowSubmit)
					assert.True(t, lang.AllowJudge)
					assert.NotEmpty(t, lang.Extensions)
				}
				assert.True(t, langNames["C++17"])
				assert.True(t, langNames["Python 3"])
				assert.True(t, langNames["Java 17"])
				assert.True(t, langNames["Go 1.21"])
				assert.True(t, langNames["Rust"])
				assert.True(t, langNames["Node.js 18"])
			},
		},
		{
			name: "list custom languages",
			setup: func(m *store.MockProblemStore) {
				m.Languages = []*pb.Language{
					{
						Id:             "custom",
						ExternalId:     "custom",
						Name:           "Custom Lang",
						TimeFactor:     1.0,
						Extensions:     []string{".custom"},
						AllowSubmit:    true,
						AllowJudge:     true,
						CompileCommand: "custom-compile",
						RunCommand:     "custom-run",
						Version:        "1.0",
					},
				}
			},
			want: func(t *testing.T, resp *pb.ListLanguagesResponse, err error) {
				require.NoError(t, err)
				assert.Len(t, resp.Languages, 1)
				assert.Equal(t, "Custom Lang", resp.Languages[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := store.NewMockProblemStore()
			tt.setup(mockStore)

			service := NewProblemService(mockStore, nil, nil)
			resp, err := service.ListLanguages(testAdminContext(), &emptypb.Empty{})

			tt.want(t, resp, err)
		})
	}
}
