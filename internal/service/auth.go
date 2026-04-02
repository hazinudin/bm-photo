package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/bina-marga/survey-photo/internal/repository"
)

// Scope constants
const (
	ScopeRead  = "read"
	ScopeWrite = "write"
	ScopeAdmin = "admin"
)

// AuthServiceImpl implements AuthService for API key validation
type AuthServiceImpl struct {
	apiKeyRepo repository.APIKeyRepository
	logger     Logger
}

// NewAuthService creates a new AuthService instance
func NewAuthService(
	apiKeyRepo repository.APIKeyRepository,
	logger Logger,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		apiKeyRepo: apiKeyRepo,
		logger:     logger,
	}
}

// ValidateAPIKey validates an API key and returns the associated key record
func (s *AuthServiceImpl) ValidateAPIKey(ctx context.Context, key string) (*repository.APIKey, error) {
	if key == "" {
		s.logger.Warn("Empty API key provided", nil)
		return nil, ErrAPIKeyInvalid
	}

	// Hash the API key to look up in database
	// API keys are stored as SHA-256 hashes
	keyHash := HashAPIKey(key)

	// Look up API key by hash
	apiKey, err := s.apiKeyRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		if errors.Is(err, repository.ErrAPIKeyNotFound) {
			s.logger.Warn("API key not found", map[string]interface{}{
				"key_hash": keyHash[:8] + "...", // Only log first 8 chars for debugging
			})
			return nil, ErrAPIKeyInvalid
		}
		s.logger.Error("Failed to lookup API key", err, nil)
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to validate API key", err)
	}

	// Check if API key is active
	if !apiKey.IsActive {
		s.logger.Warn("API key is inactive", map[string]interface{}{
			"key_id": apiKey.KeyID,
		})
		return nil, ErrAPIKeyInactive
	}

	// Check if API key has expired
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		s.logger.Warn("API key has expired", map[string]interface{}{
			"key_id":     apiKey.KeyID,
			"expires_at": apiKey.ExpiresAt.Format(time.RFC3339),
		})
		return nil, ErrAPIKeyExpired
	}

	// Update last used timestamp (fire and forget)
	go func() {
		if err := s.apiKeyRepo.UpdateLastUsed(context.Background(), apiKey.KeyID); err != nil {
			s.logger.Warn("Failed to update API key last used time", map[string]interface{}{
				"key_id": apiKey.KeyID,
				"error":  err.Error(),
			})
		}
	}()

	s.logger.Debug("API key validated", map[string]interface{}{
		"key_id": apiKey.KeyID,
		"scopes": apiKey.Scopes,
	})

	return apiKey, nil
}

// CheckScope verifies that the API key has the required scope
func (s *AuthServiceImpl) CheckScope(apiKey *repository.APIKey, scope string) error {
	if apiKey == nil {
		return ErrAPIKeyInvalid
	}

	if !hasScope(apiKey.Scopes, scope) {
		s.logger.Warn("API key missing required scope", map[string]interface{}{
			"key_id":         apiKey.KeyID,
			"required_scope": scope,
			"available":      apiKey.Scopes,
		})
		return ErrScopeNotFound
	}

	return nil
}

// CheckReadScope verifies that the API key has read scope
func (s *AuthServiceImpl) CheckReadScope(apiKey *repository.APIKey) error {
	return s.CheckScope(apiKey, ScopeRead)
}

// CheckWriteScope verifies that the API key has write scope
func (s *AuthServiceImpl) CheckWriteScope(apiKey *repository.APIKey) error {
	return s.CheckScope(apiKey, ScopeWrite)
}

// CheckAdminScope verifies that the API key has admin scope
func (s *AuthServiceImpl) CheckAdminScope(apiKey *repository.APIKey) error {
	return s.CheckScope(apiKey, ScopeAdmin)
}

// hasScope checks if the scopes slice contains the required scope
func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required {
			return true
		}
	}
	return false
}

// HashAPIKey creates a SHA-256 hash of an API key
// This is exported so it can be reused by other services (e.g., admin service)
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// APIKeyInfo contains sanitized API key information for logging
type APIKeyInfo struct {
	KeyID     string
	Scopes    []string
	ExpiresAt *time.Time
}

// Sanitize returns a safe version of the API key info for logging
func (info *APIKeyInfo) Sanitize() map[string]interface{} {
	return map[string]interface{}{
		"key_id":     info.KeyID,
		"scopes":     info.Scopes,
		"expires_at": info.ExpiresAt,
	}
}
