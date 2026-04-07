package store

import (
	"context"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executable represents a validator or run script binary
type Executable struct {
	ID             string
	ExternalID     string
	Type           string // "run", "compare", "compile"
	Description    string
	ExecutablePath string
	MD5Sum         string
	BinaryData     []byte
	CreatedAt      string
	UpdatedAt      string
}

// ExecutableStoreInterface defines the interface for executable storage
type ExecutableStoreInterface interface {
	GetByID(ctx context.Context, id string) (*Executable, error)
	GetByExternalID(ctx context.Context, externalID string) (*Executable, error)
	GetWithBinary(ctx context.Context, id string) (*Executable, error)
	Create(ctx context.Context, exec *Executable) (string, error)
	Update(ctx context.Context, id string, exec *Executable) error
	Delete(ctx context.Context, id string) error
	ListByType(ctx context.Context, execType string) ([]*Executable, error)
}

// ExecutableStore implements ExecutableStoreInterface using PostgreSQL
type ExecutableStore struct {
	db *pgxpool.Pool
}

// NewExecutableStore creates a new executable store
func NewExecutableStore(db *pgxpool.Pool) *ExecutableStore {
	return &ExecutableStore{db: db}
}

// GetByID retrieves an executable by its UUID (without binary data)
func (s *ExecutableStore) GetByID(ctx context.Context, id string) (*Executable, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, external_id, type, description, executable_path, md5sum, created_at, updated_at
		FROM executables
		WHERE id = $1
	`

	var exec Executable
	var execID pgtype.UUID
	var description, executablePath, md5sum pgtype.Text
	var createdAt, updatedAt pgtype.Timestamp

	err = s.db.QueryRow(ctx, query, parsedID).Scan(
		&execID, &exec.ExternalID, &exec.Type, &description,
		&executablePath, &md5sum, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if execID.Valid {
		exec.ID = uuid.UUID(execID.Bytes).String()
	}
	if description.Valid {
		exec.Description = description.String
	}
	if executablePath.Valid {
		exec.ExecutablePath = executablePath.String
	}
	if md5sum.Valid {
		exec.MD5Sum = md5sum.String
	}
	if createdAt.Valid {
		exec.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if updatedAt.Valid {
		exec.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return &exec, nil
}

// GetByExternalID retrieves an executable by its external ID (without binary data)
func (s *ExecutableStore) GetByExternalID(ctx context.Context, externalID string) (*Executable, error) {
	query := `
		SELECT id, external_id, type, description, executable_path, md5sum, created_at, updated_at
		FROM executables
		WHERE external_id = $1
	`

	var exec Executable
	var execID pgtype.UUID
	var description, executablePath, md5sum pgtype.Text
	var createdAt, updatedAt pgtype.Timestamp

	err := s.db.QueryRow(ctx, query, externalID).Scan(
		&execID, &exec.ExternalID, &exec.Type, &description,
		&executablePath, &md5sum, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if execID.Valid {
		exec.ID = uuid.UUID(execID.Bytes).String()
	}
	if description.Valid {
		exec.Description = description.String
	}
	if executablePath.Valid {
		exec.ExecutablePath = executablePath.String
	}
	if md5sum.Valid {
		exec.MD5Sum = md5sum.String
	}
	if createdAt.Valid {
		exec.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}
	if updatedAt.Valid {
		exec.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return &exec, nil
}

// GetWithBinary retrieves an executable including its binary data
// The binary data is stored in object storage referenced by executable_path
func (s *ExecutableStore) GetWithBinary(ctx context.Context, id string) (*Executable, error) {
	// First get the executable metadata
	exec, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// If the executable has a path, we need to fetch binary from storage
	// For now, we'll leave BinaryData empty - the service layer will handle storage fetching
	return exec, nil
}

// Create creates a new executable record
func (s *ExecutableStore) Create(ctx context.Context, exec *Executable) (string, error) {
	query := `
		INSERT INTO executables (external_id, type, description, executable_path, md5sum)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var execID pgtype.UUID
	err := s.db.QueryRow(ctx, query,
		exec.ExternalID, exec.Type, exec.Description,
		exec.ExecutablePath, exec.MD5Sum,
	).Scan(&execID)
	if err != nil {
		return "", err
	}

	if execID.Valid {
		return uuid.UUID(execID.Bytes).String(), nil
	}
	return "", nil
}

// Update updates an executable record
func (s *ExecutableStore) Update(ctx context.Context, id string, exec *Executable) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	query := `
		UPDATE executables
		SET external_id = $2, type = $3, description = $4, executable_path = $5, md5sum = $6, updated_at = NOW()
		WHERE id = $1
	`

	_, err = s.db.Exec(ctx, query,
		parsedID, exec.ExternalID, exec.Type, exec.Description,
		exec.ExecutablePath, exec.MD5Sum,
	)
	return err
}

// Delete deletes an executable record
func (s *ExecutableStore) Delete(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, "DELETE FROM executables WHERE id = $1", parsedID)
	return err
}

// ListByType lists all executables of a given type
func (s *ExecutableStore) ListByType(ctx context.Context, execType string) ([]*Executable, error) {
	query := `
		SELECT id, external_id, type, description, executable_path, md5sum, created_at, updated_at
		FROM executables
		WHERE type = $1
		ORDER BY external_id
	`

	rows, err := s.db.Query(ctx, query, execType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executables []*Executable
	for rows.Next() {
		var exec Executable
		var execID pgtype.UUID
		var description, executablePath, md5sum pgtype.Text
		var createdAt, updatedAt pgtype.Timestamp

		err := rows.Scan(
			&execID, &exec.ExternalID, &exec.Type, &description,
			&executablePath, &md5sum, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if execID.Valid {
			exec.ID = uuid.UUID(execID.Bytes).String()
		}
		if description.Valid {
			exec.Description = description.String
		}
		if executablePath.Valid {
			exec.ExecutablePath = executablePath.String
		}
		if md5sum.Valid {
			exec.MD5Sum = md5sum.String
		}
		if createdAt.Valid {
			exec.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if updatedAt.Valid {
			exec.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}

		executables = append(executables, &exec)
	}

	return executables, nil
}

// ExecutableResponse is the JSON response format for executable API
type ExecutableResponse struct {
	ID             string `json:"id"`
	ExternalID     string `json:"external_id"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	ExecutablePath string `json:"executable_path"`
	MD5Sum         string `json:"md5sum"`
	BinaryData     string `json:"binary_data,omitempty"` // Base64 encoded
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ToResponse converts Executable to ExecutableResponse with optional binary data
func (e *Executable) ToResponse(includeBinary bool) *ExecutableResponse {
	resp := &ExecutableResponse{
		ID:             e.ID,
		ExternalID:     e.ExternalID,
		Type:           e.Type,
		Description:    e.Description,
		ExecutablePath: e.ExecutablePath,
		MD5Sum:         e.MD5Sum,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}

	if includeBinary && len(e.BinaryData) > 0 {
		resp.BinaryData = base64.StdEncoding.EncodeToString(e.BinaryData)
	}

	return resp
}
