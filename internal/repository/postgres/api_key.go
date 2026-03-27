package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/bina-marga/survey-photo/internal/repository"
)

// APIKeyRepository implements repository.APIKeyRepository
type APIKeyRepository struct {
	db *PostgresDB
}

// NewAPIKeyRepository creates a new APIKeyRepository
func NewAPIKeyRepository(db *PostgresDB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// apiKeyRow mirrors DB schema
type apiKeyRow struct {
	KeyID       string
	KeyHash     string
	Scopes      []byte // JSON array
	Description *string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	IsActive    bool
}

func (r *APIKeyRepository) Create(ctx context.Context, apiKey *repository.APIKey) error {
	query := `
		INSERT INTO api_keys (key_id, key_hash, scopes, description, created_at, expires_at, last_used_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	scopesJSON, err := json.Marshal(apiKey.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	_, err = r.db.Pool().Exec(ctx, query,
		apiKey.KeyID,
		apiKey.KeyHash,
		scopesJSON,
		apiKey.Description,
		apiKey.CreatedAt,
		apiKey.ExpiresAt,
		apiKey.LastUsedAt,
		apiKey.IsActive,
	)
	if err != nil {
		return fmt.Errorf("failed to insert api key: %w", err)
	}

	return nil
}

func (r *APIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*repository.APIKey, error) {
	query := `
		SELECT key_id, key_hash, scopes, description, created_at, expires_at, last_used_at, is_active
		FROM api_keys
		WHERE key_hash = $1
	`

	row := apiKeyRow{}
	err := r.db.Pool().QueryRow(ctx, query, keyHash).Scan(
		&row.KeyID,
		&row.KeyHash,
		&row.Scopes,
		&row.Description,
		&row.CreatedAt,
		&row.ExpiresAt,
		&row.LastUsedAt,
		&row.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key by hash: %w", err)
	}

	return r.rowToAPIKey(&row)
}

func (r *APIKeyRepository) GetByID(ctx context.Context, keyID string) (*repository.APIKey, error) {
	query := `
		SELECT key_id, key_hash, scopes, description, created_at, expires_at, last_used_at, is_active
		FROM api_keys
		WHERE key_id = $1
	`

	row := apiKeyRow{}
	err := r.db.Pool().QueryRow(ctx, query, keyID).Scan(
		&row.KeyID,
		&row.KeyHash,
		&row.Scopes,
		&row.Description,
		&row.CreatedAt,
		&row.ExpiresAt,
		&row.LastUsedAt,
		&row.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to get api key by id: %w", err)
	}

	return r.rowToAPIKey(&row)
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, keyID string) error {
	query := `
		UPDATE api_keys
		SET last_used_at = NOW()
		WHERE key_id = $1
	`

	result, err := r.db.Pool().Exec(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrAPIKeyNotFound
	}

	return nil
}

func (r *APIKeyRepository) Revoke(ctx context.Context, keyID string) error {
	query := `
		UPDATE api_keys
		SET is_active = false
		WHERE key_id = $1
	`

	result, err := r.db.Pool().Exec(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrAPIKeyNotFound
	}

	return nil
}

func (r *APIKeyRepository) List(ctx context.Context, activeOnly bool) ([]*repository.APIKey, error) {
	var query string
	var args []interface{}

	if activeOnly {
		query = `
			SELECT key_id, key_hash, scopes, description, created_at, expires_at, last_used_at, is_active
			FROM api_keys
			WHERE is_active = true
			ORDER BY created_at DESC
		`
	} else {
		query = `
			SELECT key_id, key_hash, scopes, description, created_at, expires_at, last_used_at, is_active
			FROM api_keys
			ORDER BY created_at DESC
		`
	}

	rows, err := r.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*repository.APIKey
	for rows.Next() {
		row := apiKeyRow{}
		err := rows.Scan(
			&row.KeyID,
			&row.KeyHash,
			&row.Scopes,
			&row.Description,
			&row.CreatedAt,
			&row.ExpiresAt,
			&row.LastUsedAt,
			&row.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key row: %w", err)
		}

		apiKey, err := r.rowToAPIKey(&row)
		if err != nil {
			return nil, err
		}
		apiKeys = append(apiKeys, apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating api key rows: %w", err)
	}

	return apiKeys, nil
}

func (r *APIKeyRepository) Delete(ctx context.Context, keyID string) error {
	query := `DELETE FROM api_keys WHERE key_id = $1`

	result, err := r.db.Pool().Exec(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrAPIKeyNotFound
	}

	return nil
}

func (r *APIKeyRepository) rowToAPIKey(row *apiKeyRow) (*repository.APIKey, error) {
	var scopes []string
	if row.Scopes != nil {
		if err := json.Unmarshal(row.Scopes, &scopes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}
	}

	description := ""
	if row.Description != nil {
		description = *row.Description
	}

	return &repository.APIKey{
		KeyID:       row.KeyID,
		KeyHash:     row.KeyHash,
		Scopes:      scopes,
		Description: description,
		CreatedAt:   row.CreatedAt,
		ExpiresAt:   row.ExpiresAt,
		LastUsedAt:  row.LastUsedAt,
		IsActive:    row.IsActive,
	}, nil
}
