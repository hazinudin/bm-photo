package rest

import "time"

// CreateAPIKeyRequest - Request to create a new API key
type CreateAPIKeyRequest struct {
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`                    // e.g., ["read", "write"]
	ExpiresIn   *int     `json:"expires_in_days,omitempty"` // nil = no expiry
}

// CreateAPIKeyResponse - Response with the newly created API key (raw key shown once only)
type CreateAPIKeyResponse struct {
	KeyID       string     `json:"key_id"`
	APIKey      string     `json:"api_key"` // The actual key - shown ONLY ONCE
	KeyHash     string     `json:"key_hash"`
	Scopes      []string   `json:"scopes"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ListAPIKeysResponse - Response listing API keys (without raw keys)
type ListAPIKeysResponse struct {
	APIKeys []APIKeyInfo `json:"api_keys"`
}

// APIKeyInfo - Sanitized API key info (no raw key)
type APIKeyInfo struct {
	KeyID       string     `json:"key_id"`
	Scopes      []string   `json:"scopes"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	IsActive    bool       `json:"is_active"`
}

// RevokeAPIKeyResponse - Response after revoking a key
type RevokeAPIKeyResponse struct {
	KeyID   string `json:"key_id"`
	Message string `json:"message"`
}
