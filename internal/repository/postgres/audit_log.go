package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// AuditLogRepository implements repository.AuditLogRepository
type AuditLogRepository struct {
	db *PostgresDB
}

// NewAuditLogRepository creates a new AuditLogRepository
func NewAuditLogRepository(db *PostgresDB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// auditLogRow mirrors DB schema
type auditLogRow struct {
	LogID      string
	PhotoID    *string // nullable UUID as string
	Operation  string
	APIKeyID   string
	OperatedAt time.Time
	Details    []byte // JSONB
}

func (r *AuditLogRepository) Create(ctx context.Context, entry *repository.AuditLogEntry) error {
	query := `
		INSERT INTO photo_audit_log (log_id, photo_id, operation, api_key_id, operated_at, details)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	var photoID *string
	if entry.PhotoID != nil {
		pid := entry.PhotoID.String()
		photoID = &pid
	}

	var detailsJSON []byte
	var err error
	if entry.Details != nil {
		detailsJSON, err = json.Marshal(entry.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	_, err = r.db.Pool().Exec(ctx, query,
		entry.LogID,
		photoID,
		entry.Operation,
		entry.APIKeyID,
		entry.OperatedAt,
		detailsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit log entry: %w", err)
	}

	return nil
}

func (r *AuditLogRepository) GetByPhotoID(ctx context.Context, photoID vo.PhotoID, page, perPage int) ([]*repository.AuditLogEntry, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	query := `
		SELECT log_id, photo_id, operation, api_key_id, operated_at, details
		FROM photo_audit_log
		WHERE photo_id = $1
		ORDER BY operated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool().Query(ctx, query, photoID.String(), perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by photo id: %w", err)
	}
	defer rows.Close()

	return r.scanAuditLogEntries(rows)
}

func (r *AuditLogRepository) GetByAPIKey(ctx context.Context, apiKeyID string, page, perPage int) ([]*repository.AuditLogEntry, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	offset := (page - 1) * perPage

	query := `
		SELECT log_id, photo_id, operation, api_key_id, operated_at, details
		FROM photo_audit_log
		WHERE api_key_id = $1
		ORDER BY operated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool().Query(ctx, query, apiKeyID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by api key: %w", err)
	}
	defer rows.Close()

	return r.scanAuditLogEntries(rows)
}

func (r *AuditLogRepository) scanAuditLogEntries(rows pgx.Rows) ([]*repository.AuditLogEntry, error) {
	var entries []*repository.AuditLogEntry

	for rows.Next() {
		row := auditLogRow{}
		err := rows.Scan(
			&row.LogID,
			&row.PhotoID,
			&row.Operation,
			&row.APIKeyID,
			&row.OperatedAt,
			&row.Details,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log row: %w", err)
		}

		entry, err := r.rowToAuditLogEntry(&row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit log rows: %w", err)
	}

	return entries, nil
}

func (r *AuditLogRepository) rowToAuditLogEntry(row *auditLogRow) (*repository.AuditLogEntry, error) {
	var photoID *vo.PhotoID
	if row.PhotoID != nil && *row.PhotoID != "" {
		pid, err := vo.ParsePhotoID(*row.PhotoID)
		if err != nil {
			return nil, fmt.Errorf("invalid photo id in audit log: %w", err)
		}
		photoID = &pid
	}

	var details map[string]interface{}
	if len(row.Details) > 0 {
		if err := json.Unmarshal(row.Details, &details); err != nil {
			return nil, fmt.Errorf("failed to unmarshal details: %w", err)
		}
	}

	return &repository.AuditLogEntry{
		LogID:      row.LogID,
		PhotoID:    photoID,
		Operation:  row.Operation,
		APIKeyID:   row.APIKeyID,
		OperatedAt: row.OperatedAt,
		Details:    details,
	}, nil
}
