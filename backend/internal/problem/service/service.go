package service

import (
	"context"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/emptypb"

	commonv1 "github.com/online-judge/gen/go/common/v1"
	pb "github.com/online-judge/gen/go/problem/v1"
	"github.com/online-judge/backend/internal/problem/store"
)

type ProblemService struct {
	pb.UnimplementedProblemServiceServer
	store store.ProblemStoreInterface
	cache *redis.Client
}

func NewProblemService(s store.ProblemStoreInterface, cache *redis.Client) *ProblemService {
	return &ProblemService{
		store: s,
		cache: cache,
	}
}

func (s *ProblemService) ListProblems(ctx context.Context, req *pb.ListProblemsRequest) (*pb.ListProblemsResponse, error) {
	problems, total, err := s.store.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.ListProblemsResponse{
		Problems: problems,
		Pagination: &commonv1.PaginatedResponse{
			Total:    total,
			Page:     req.GetPagination().GetPage(),
			PageSize: req.GetPagination().GetPageSize(),
		},
	}, nil
}

func (s *ProblemService) GetProblem(ctx context.Context, req *pb.GetProblemRequest) (*pb.GetProblemResponse, error) {
	problem, err := s.store.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// Get sample test cases
	samples, err := s.store.ListTestCases(ctx, req.GetId(), true)
	if err != nil {
		return nil, err
	}

	return &pb.GetProblemResponse{
		Problem:         problem,
		SampleTestCases: samples,
	}, nil
}

func (s *ProblemService) CreateProblem(ctx context.Context, req *pb.CreateProblemRequest) (*pb.CreateProblemResponse, error) {
	id, err := s.store.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.CreateProblemResponse{Id: id}, nil
}

func (s *ProblemService) UpdateProblem(ctx context.Context, req *pb.UpdateProblemRequest) (*pb.UpdateProblemResponse, error) {
	if err := s.store.Update(ctx, req.GetId(), req); err != nil {
		return nil, err
	}

	problem, err := s.store.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.UpdateProblemResponse{Problem: problem}, nil
}

func (s *ProblemService) DeleteProblem(ctx context.Context, req *pb.DeleteProblemRequest) (*emptypb.Empty, error) {
	if err := s.store.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *ProblemService) ListTestCases(ctx context.Context, req *pb.ListTestCasesRequest) (*pb.ListTestCasesResponse, error) {
	testCases, err := s.store.ListTestCases(ctx, req.GetProblemId(), req.GetSamplesOnly())
	if err != nil {
		return nil, err
	}

	return &pb.ListTestCasesResponse{TestCases: testCases}, nil
}

func (s *ProblemService) CreateTestCase(ctx context.Context, req *pb.CreateTestCaseRequest) (*pb.CreateTestCaseResponse, error) {
	id, inputPath, outputPath, err := s.store.CreateTestCase(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.CreateTestCaseResponse{
		Id:         id,
		InputPath:  inputPath,
		OutputPath: outputPath,
	}, nil
}

func (s *ProblemService) UpdateTestCase(ctx context.Context, req *pb.UpdateTestCaseRequest) (*pb.UpdateTestCaseResponse, error) {
	tc, err := s.store.UpdateTestCase(ctx, req.GetId(), req)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateTestCaseResponse{TestCase: tc}, nil
}

func (s *ProblemService) DeleteTestCase(ctx context.Context, req *pb.DeleteTestCaseRequest) (*emptypb.Empty, error) {
	if err := s.store.DeleteTestCase(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *ProblemService) BatchUploadTestCases(ctx context.Context, req *pb.BatchUploadTestCasesRequest) (*pb.BatchUploadTestCasesResponse, error) {
	testCases, err := s.store.BatchCreateTestCases(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.BatchUploadTestCasesResponse{TestCases: testCases}, nil
}

func (s *ProblemService) ListLanguages(ctx context.Context, req *emptypb.Empty) (*pb.ListLanguagesResponse, error) {
	languages, err := s.store.ListLanguages(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.ListLanguagesResponse{Languages: languages}, nil
}

func (s *ProblemService) GetProblemStatement(ctx context.Context, req *pb.GetProblemStatementRequest) (*pb.GetProblemStatementResponse, error) {
	statement, err := s.store.GetProblemStatement(ctx, req.GetProblemId(), req.GetLanguage())
	if err != nil {
		return nil, err
	}

	return &pb.GetProblemStatementResponse{Statement: statement}, nil
}

func (s *ProblemService) SetProblemStatement(ctx context.Context, req *pb.SetProblemStatementRequest) (*pb.SetProblemStatementResponse, error) {
	statement, err := s.store.SetProblemStatement(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.SetProblemStatementResponse{Statement: statement}, nil
}
