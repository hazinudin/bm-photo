package service

import (
	"context"

	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// UploadService handles two-phase upload workflow
type UploadService interface {
	// GetSignedURL Phase 1: Validates request, creates photo record,
	// generates signed URL for GCS upload.
	// Returns upload token, signed URL, photo ID, and expiry time.
	GetSignedURL(ctx context.Context, req *rest.GetSignedUploadURLRequest, apiKeyID string) (*rest.GetSignedUploadURLResponse, error)

	// ConfirmUpload Phase 2: Validates token, verifies file in GCS,
	// marks upload as completed.
	ConfirmUpload(ctx context.Context, token vo.UploadToken, apiKeyID string) (*rest.ConfirmUploadResponse, error)
}

// PhotoService handles photo CRUD and browsing operations
type PhotoService interface {
	// GetByID retrieves a photo by its ID
	GetByID(ctx context.Context, id vo.PhotoID) (*entity.Photo, error)

	// Browse retrieves photos with optional filters (route, STA, lane)
	Browse(ctx context.Context, filter repository.BrowseFilter) (*rest.BrowsePhotosResponse, error)

	// Search performs advanced search with multiple filters
	Search(ctx context.Context, filter repository.SearchFilter) (*rest.SearchPhotosResponse, error)

	// Update modifies photo metadata (description, tags, lane)
	Update(ctx context.Context, id vo.PhotoID, req *rest.UpdatePhotoRequest) (*rest.UpdatePhotoResponse, error)

	// Delete soft-deletes or hard-deletes a photo
	Delete(ctx context.Context, id vo.PhotoID, hard bool, apiKeyID string) (*rest.DeletePhotoResponse, error)
}

// AuthService handles API key validation and authentication
type AuthService interface {
	// ValidateAPIKey validates an API key and returns the associated key record
	ValidateAPIKey(ctx context.Context, key string) (*repository.APIKey, error)

	// CheckScope verifies that the API key has the required scope
	CheckScope(apiKey *repository.APIKey, scope string) error
}

// AuditLogService tracks operations for auditing purposes
type AuditLogService interface {
	// LogUploadRequest logs a signed URL generation request
	LogUploadRequest(ctx context.Context, photoID, apiKeyID string, details map[string]interface{})

	// LogUploadConfirm logs an upload confirmation
	LogUploadConfirm(ctx context.Context, photoID, apiKeyID string)

	// LogPhotoUpdate logs a photo metadata update
	LogPhotoUpdate(ctx context.Context, photoID, apiKeyID string, details map[string]interface{})

	// LogPhotoDelete logs a photo deletion
	LogPhotoDelete(ctx context.Context, photoID, apiKeyID string, hard bool)
}

// GCSClient defines the interface for Google Cloud Storage operations
// This will be implemented by the client layer
type GCSClient interface {
	// GenerateSignedURL generates a signed URL for uploading a file
	GenerateSignedURL(objectName string, contentType string, expiryMinutes int) (string, error)

	// FileExists checks if a file exists in GCS
	FileExists(objectName string) (bool, error)

	// DeleteFile deletes a file from GCS
	DeleteFile(objectName string) error
}

// Logger defines the logging interface used by services
type Logger interface {
	// Info logs an informational message
	Info(msg string, ctx ...map[string]interface{})

	// Error logs an error
	Error(msg string, err error, ctx ...map[string]interface{})

	// Warn logs a warning
	Warn(msg string, ctx ...map[string]interface{})

	// Debug logs a debug message
	Debug(msg string, ctx ...map[string]interface{})
}
