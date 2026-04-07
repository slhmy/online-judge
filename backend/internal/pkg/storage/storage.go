package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageBackend string

const (
	StorageMinIO StorageBackend = "minio"
	StorageS3    StorageBackend = "s3"
	StorageLocal StorageBackend = "local"
)

type StorageService struct {
	client    *minio.Client
	bucket    string
	backend   StorageBackend
	localPath string
}

type Config struct {
	Backend   StorageBackend
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	LocalPath string
}

func NewStorageService(cfg *Config) (*StorageService, error) {
	if cfg.Backend == StorageLocal {
		return &StorageService{
			backend:   StorageLocal,
			localPath: cfg.LocalPath,
		}, nil
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: true,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}

	// Create bucket if not exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, err
		}
	}

	return &StorageService{
		client:  client,
		bucket:  cfg.Bucket,
		backend: cfg.Backend,
	}, nil
}

func (s *StorageService) Upload(ctx context.Context, path string, reader io.Reader, size int64, contentType string) error {
	if s.backend == StorageLocal {
		return s.uploadLocal(path, reader, size)
	}

	_, err := s.client.PutObject(ctx, s.bucket, path, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *StorageService) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	if s.backend == StorageLocal {
		return s.downloadLocal(path)
	}

	return s.client.GetObject(ctx, s.bucket, path, minio.GetObjectOptions{})
}

func (s *StorageService) Delete(ctx context.Context, path string) error {
	if s.backend == StorageLocal {
		return s.deleteLocal(path)
	}

	return s.client.RemoveObject(ctx, s.bucket, path, minio.RemoveObjectOptions{})
}

func (s *StorageService) Exists(ctx context.Context, path string) (bool, error) {
	if s.backend == StorageLocal {
		return s.existsLocal(path)
	}

	_, err := s.client.StatObject(ctx, s.bucket, path, minio.StatObjectOptions{})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// Path helpers
const (
	PathTestcaseInput    = "testcases/{problem_id}/{testcase_id}/input"
	PathTestcaseOutput   = "testcases/{problem_id}/{testcase_id}/output"
	PathSubmissionSource = "submissions/{submission_id}/source"
	PathSubmissionBinary = "submissions/{submission_id}/binary"
	PathJudgingOutput    = "judgings/{judging_id}/output"
	PathJudgingError     = "judgings/{judging_id}/error"
	PathJudgingDiff      = "judgings/{judging_id}/diff"
)
