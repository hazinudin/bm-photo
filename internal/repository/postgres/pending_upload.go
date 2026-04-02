package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// PendingUploadRepository implements repository.PendingUploadRepository
type PendingUploadRepository struct {
	db *PostgresDB
}

// NewPendingUploadRepository creates a new PendingUploadRepository
func NewPendingUploadRepository(db *PostgresDB) *PendingUploadRepository {
	return &PendingUploadRepository{db: db}
}

// pendingUploadRow mirrors the DB schema for scanning
type pendingUploadRow struct {
	UploadToken string
	PhotoID     string
	APIKeyID    string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Status      string
}

func (r *pendingUploadRow) toPendingUpload() (*repository.PendingUpload, error) {
	photoID, err := vo.ParsePhotoID(r.PhotoID)
	if err != nil {
		return nil, fmt.Errorf("invalid photo ID: %w", err)
	}

	uploadToken, err := vo.ParseUploadToken(r.UploadToken)
	if err != nil {
		return nil, fmt.Errorf("invalid upload token: %w", err)
	}

	uploadStatus, err := vo.ParseUploadStatus(r.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid upload status: %w", err)
	}

	return &repository.PendingUpload{
		UploadToken: uploadToken,
		PhotoID:     photoID,
		APIKeyID:    r.APIKeyID,
		CreatedAt:   r.CreatedAt,
		ExpiresAt:   r.ExpiresAt,
		Status:      uploadStatus,
	}, nil
}

func (r *PendingUploadRepository) Create(ctx context.Context, upload *repository.PendingUpload) error {
	query := `
		INSERT INTO pending_uploads (
			upload_token, photo_id, api_key_id,
			created_at, expires_at,
			status
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6
		)`

	_, err := r.db.Pool().Exec(ctx, query,
		upload.UploadToken.String(),
		upload.PhotoID.String(),
		upload.APIKeyID,
		upload.CreatedAt,
		upload.ExpiresAt,
		upload.Status.String(),
	)

	if err != nil {
		return fmt.Errorf("failed to create pending upload: %w", err)
	}

	return nil
}

func (r *PendingUploadRepository) GetByToken(ctx context.Context, token vo.UploadToken) (*repository.PendingUpload, error) {
	query := `
		SELECT upload_token, photo_id, api_key_id,
			created_at, expires_at,
			status
		FROM pending_uploads
		WHERE upload_token = $1`

	row := r.db.Pool().QueryRow(ctx, query, token.String())

	return r.scanPendingUpload(row)
}

func (r *PendingUploadRepository) GetByPhotoID(ctx context.Context, photoID vo.PhotoID) (*repository.PendingUpload, error) {
	query := `
		SELECT upload_token, photo_id, api_key_id,
			created_at, expires_at,
			status
		FROM pending_uploads
		WHERE photo_id = $1`

	row := r.db.Pool().QueryRow(ctx, query, photoID.String())

	return r.scanPendingUpload(row)
}

func (r *PendingUploadRepository) MarkAsCompleted(ctx context.Context, token vo.UploadToken) error {
	query := `
		UPDATE pending_uploads SET
			status = $2
		WHERE upload_token = $1`

	result, err := r.db.Pool().Exec(ctx, query, token.String(), vo.UploadStatusCompleted.String())
	if err != nil {
		return fmt.Errorf("failed to mark upload as completed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrTokenNotFound
	}

	return nil
}

func (r *PendingUploadRepository) MarkAsExpired(ctx context.Context, before time.Time) (int64, error) {
	query := `
		UPDATE pending_uploads SET
			status = $2
		WHERE expires_at < $1 AND status = 'pending'`

	result, err := r.db.Pool().Exec(ctx, query, before, vo.UploadStatusExpired.String())
	if err != nil {
		return 0, fmt.Errorf("failed to mark uploads as expired: %w", err)
	}

	return result.RowsAffected(), nil
}

func (r *PendingUploadRepository) Delete(ctx context.Context, token vo.UploadToken) error {
	query := `DELETE FROM pending_uploads WHERE upload_token = $1`

	result, err := r.db.Pool().Exec(ctx, query, token.String())
	if err != nil {
		return fmt.Errorf("failed to delete pending upload: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrTokenNotFound
	}

	return nil
}

func (r *PendingUploadRepository) DeleteByPhotoID(ctx context.Context, photoID vo.PhotoID) error {
	query := `DELETE FROM pending_uploads WHERE photo_id = $1`

	_, err := r.db.Pool().Exec(ctx, query, photoID.String())
	if err != nil {
		return fmt.Errorf("failed to delete pending uploads by photo ID: %w", err)
	}

	return nil
}

func (r *PendingUploadRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM pending_uploads WHERE expires_at < $1 AND status = 'expired'`

	result, err := r.db.Pool().Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired uploads: %w", err)
	}

	return result.RowsAffected(), nil
}

func (r *PendingUploadRepository) CountActiveByAPIKey(ctx context.Context, apiKeyID string) (int64, error) {
	query := `
		SELECT COUNT(*) FROM pending_uploads
		WHERE api_key_id = $1 AND status = 'pending'`

	var count int64
	if err := r.db.Pool().QueryRow(ctx, query, apiKeyID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count active uploads: %w", err)
	}

	return count, nil
}

func (r *PendingUploadRepository) GetExpired(ctx context.Context, before time.Time) ([]*repository.PendingUpload, error) {
	query := `
		SELECT upload_token, photo_id, api_key_id,
			created_at, expires_at,
			status
		FROM pending_uploads
		WHERE expires_at < $1 AND status = 'pending'
		ORDER BY created_at`

	rows, err := r.db.Pool().Query(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired uploads: %w", err)
	}
	defer rows.Close()

	return r.scanPendingUploads(rows)
}

func (r *PendingUploadRepository) ExpireTokensByPhotoID(ctx context.Context, photoID vo.PhotoID) error {
	query := `
		UPDATE pending_uploads SET
			status = $2
		WHERE photo_id = $1 AND status = 'pending'`

	_, err := r.db.Pool().Exec(ctx, query, photoID.String(), vo.UploadStatusExpired.String())
	if err != nil {
		return fmt.Errorf("failed to expire tokens: %w", err)
	}

	return nil
}

func (r *PendingUploadRepository) scanPendingUpload(row pgx.Row) (*repository.PendingUpload, error) {
	var p pendingUploadRow
	err := row.Scan(
		&p.UploadToken,
		&p.PhotoID,
		&p.APIKeyID,
		&p.CreatedAt,
		&p.ExpiresAt,
		&p.Status,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to scan pending upload: %w", err)
	}

	return p.toPendingUpload()
}

func (r *PendingUploadRepository) scanPendingUploads(rows pgx.Rows) ([]*repository.PendingUpload, error) {
	var uploads []*repository.PendingUpload

	for rows.Next() {
		var p pendingUploadRow
		err := rows.Scan(
			&p.UploadToken,
			&p.PhotoID,
			&p.APIKeyID,
			&p.CreatedAt,
			&p.ExpiresAt,
			&p.Status,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan pending upload row: %w", err)
		}

		upload, err := p.toPendingUpload()
		if err != nil {
			return nil, err
		}

		uploads = append(uploads, upload)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending upload rows: %w", err)
	}

	return uploads, nil
}
