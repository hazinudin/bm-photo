package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// AdminService handles admin operations for API key management
type AdminService interface {
	CreateAPIKey(ctx context.Context, req *rest.CreateAPIKeyRequest) (*rest.CreateAPIKeyResponse, error)
	ListAPIKeys(ctx context.Context, activeOnly bool) (*rest.ListAPIKeysResponse, error)
	RevokeAPIKey(ctx context.Context, keyID string) (*rest.RevokeAPIKeyResponse, error)
}

type AdminServiceImpl struct {
	apiKeyRepo repository.APIKeyRepository
	logger     Logger
}

func NewAdminService(apiKeyRepo repository.APIKeyRepository, logger Logger) *AdminServiceImpl {
	return &AdminServiceImpl{
		apiKeyRepo: apiKeyRepo,
		logger:     logger,
	}
}

// generateAPIKey generates a new random API key (64 hex characters = 32 bytes)
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// generateUUID generates a UUID v4-like identifier
func generateUUID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	// Set version 4 and variant bits
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:])
}

func (s *AdminServiceImpl) CreateAPIKey(ctx context.Context, req *rest.CreateAPIKeyRequest) (*rest.CreateAPIKeyResponse, error) {
	// Validate scopes
	validScopes := map[string]bool{"read": true, "write": true, "admin": true}
	for _, scope := range req.Scopes {
		if !validScopes[scope] {
			return nil, fmt.Errorf("%w: %s", ErrInvalidScope, scope)
		}
	}

	// Generate new API key
	rawKey, err := generateAPIKey()
	if err != nil {
		s.logger.Error("failed to generate API key", err, nil)
		return nil, ErrAPIKeyCreationFailed
	}

	keyHash := HashAPIKey(rawKey)
	keyID := generateUUID()

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &t
	}

	createdAt := time.Now()

	apiKey := &repository.APIKey{
		KeyID:       keyID,
		KeyHash:     keyHash,
		Scopes:      req.Scopes,
		Description: req.Description,
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
		IsActive:    true,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		s.logger.Error("failed to create API key in repository", err, nil)
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	s.logger.Info("API key created", map[string]interface{}{
		"key_id": keyID,
		"scopes": strings.Join(req.Scopes, ","),
	})

	return &rest.CreateAPIKeyResponse{
		KeyID:       keyID,
		APIKey:      rawKey, // Show raw key ONLY HERE
		KeyHash:     keyHash,
		Scopes:      req.Scopes,
		Description: req.Description,
		ExpiresAt:   expiresAt,
		CreatedAt:   createdAt,
	}, nil
}

func (s *AdminServiceImpl) ListAPIKeys(ctx context.Context, activeOnly bool) (*rest.ListAPIKeysResponse, error) {
	apiKeys, err := s.apiKeyRepo.List(ctx, activeOnly)
	if err != nil {
		s.logger.Error("failed to list API keys", err, nil)
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	infos := make([]rest.APIKeyInfo, len(apiKeys))
	for i, key := range apiKeys {
		infos[i] = rest.APIKeyInfo{
			KeyID:       key.KeyID,
			Scopes:      key.Scopes,
			Description: key.Description,
			CreatedAt:   key.CreatedAt,
			ExpiresAt:   key.ExpiresAt,
			LastUsedAt:  key.LastUsedAt,
			IsActive:    key.IsActive,
		}
	}

	return &rest.ListAPIKeysResponse{APIKeys: infos}, nil
}

func (s *AdminServiceImpl) RevokeAPIKey(ctx context.Context, keyID string) (*rest.RevokeAPIKeyResponse, error) {
	if err := s.apiKeyRepo.Revoke(ctx, keyID); err != nil {
		if errors.Is(err, repository.ErrAPIKeyNotFound) {
			return nil, ErrAPIKeyNotFound
		}
		s.logger.Error("failed to revoke API key", err, map[string]interface{}{
			"key_id": keyID,
		})
		return nil, fmt.Errorf("failed to revoke API key: %w", err)
	}

	s.logger.Info("API key revoked", map[string]interface{}{
		"key_id": keyID,
	})

	return &rest.RevokeAPIKeyResponse{
		KeyID:   keyID,
		Message: "API key revoked successfully",
	}, nil
}
