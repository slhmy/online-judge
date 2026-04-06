package service

import (
	"context"

	"github.com/online-judge/backend/internal/problem/store"
)

// ExecutableService handles executable management
type ExecutableService struct {
	store  store.ExecutableStoreInterface
	storage StorageFetcher
}

// StorageFetcher interface for fetching binary data from object storage
type StorageFetcher interface {
	Fetch(ctx context.Context, path string) ([]byte, error)
}

// NewExecutableService creates a new executable service
func NewExecutableService(s store.ExecutableStoreInterface, storage StorageFetcher) *ExecutableService {
	return &ExecutableService{
		store:  s,
		storage: storage,
	}
}

// GetExecutable retrieves an executable by ID (without binary data)
func (s *ExecutableService) GetExecutable(ctx context.Context, id string) (*store.Executable, error) {
	return s.store.GetByID(ctx, id)
}

// GetExecutableByExternalID retrieves an executable by external ID
func (s *ExecutableService) GetExecutableByExternalID(ctx context.Context, externalID string) (*store.Executable, error) {
	return s.store.GetByExternalID(ctx, externalID)
}

// GetExecutableWithBinary retrieves an executable with its binary data loaded
func (s *ExecutableService) GetExecutableWithBinary(ctx context.Context, id string) (*store.Executable, error) {
	exec, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Fetch binary data from object storage if path is set
	if exec.ExecutablePath != "" && s.storage != nil {
		binaryData, err := s.storage.Fetch(ctx, exec.ExecutablePath)
		if err != nil {
			// Log error but return executable without binary
			// The judge system will handle the missing binary appropriately
			return exec, nil
		}
		exec.BinaryData = binaryData
	}

	return exec, nil
}

// CreateExecutable creates a new executable
func (s *ExecutableService) CreateExecutable(ctx context.Context, exec *store.Executable) (string, error) {
	return s.store.Create(ctx, exec)
}

// UpdateExecutable updates an existing executable
func (s *ExecutableService) UpdateExecutable(ctx context.Context, id string, exec *store.Executable) error {
	return s.store.Update(ctx, id, exec)
}

// DeleteExecutable deletes an executable
func (s *ExecutableService) DeleteExecutable(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListValidators lists all validator executables (type = "compare")
func (s *ExecutableService) ListValidators(ctx context.Context) ([]*store.Executable, error) {
	return s.store.ListByType(ctx, "compare")
}

// ListRunScripts lists all run script executables (type = "run")
func (s *ExecutableService) ListRunScripts(ctx context.Context) ([]*store.Executable, error) {
	return s.store.ListByType(ctx, "run")
}

// ListCompileScripts lists all compile script executables (type = "compile")
func (s *ExecutableService) ListCompileScripts(ctx context.Context) ([]*store.Executable, error) {
	return s.store.ListByType(ctx, "compile")
}