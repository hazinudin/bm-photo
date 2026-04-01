package repository

import (
	"context"
	"time"

	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
)

// PhotoRepository defines operations for photo catalog.
type PhotoRepository interface {
	// Create inserts a new photo into the repository.
	Create(ctx context.Context, photo *entity.Photo) error

	// GetByID retrieves a photo by its ID.
	GetByID(ctx context.Context, id vo.PhotoID) (*entity.Photo, error)

	// GetByUploadToken retrieves a photo by its upload token.
	GetByUploadToken(ctx context.Context, token vo.UploadToken) (*entity.Photo, error)

	// Update modifies an existing photo in the repository.
	Update(ctx context.Context, photo *entity.Photo) error

	// SoftDelete marks a photo as deleted without removing its data.
	SoftDelete(ctx context.Context, id vo.PhotoID, deletedBy string) error

	// HardDelete permanently removes a photo from the repository.
	HardDelete(ctx context.Context, id vo.PhotoID) error

	// Restore un-deletes a soft-deleted photo.
	Restore(ctx context.Context, id vo.PhotoID) error

	// UpdateSTA updates the STA value and source for a photo.
	// Both staValue and source can be nil to set the field to NULL.
	UpdateSTA(ctx context.Context, id vo.PhotoID, staValue *float64, source *vo.STASource) error

	// Browse retrieves a paginated list of photos with optional filters.
	Browse(ctx context.Context, filter BrowseFilter) (*BrowseResult, error)

	// Search performs a search with more advanced filter options.
	Search(ctx context.Context, filter SearchFilter) (*BrowseResult, error)

	// Exists checks if a photo with the given ID exists.
	Exists(ctx context.Context, id vo.PhotoID) (bool, error)

	// UpdateUploadStatus updates the upload status of a photo.
	UpdateUploadStatus(ctx context.Context, id vo.PhotoID, status vo.UploadStatus) error

	// IncrementRetryCount increments the retry count for a photo.
	// Returns ErrRetryLimitExceeded if the maximum retry count (5) has been reached.
	IncrementRetryCount(ctx context.Context, id vo.PhotoID) error

	// FindPendingByIDAndAPIKey retrieves a pending photo by ID and verifies API key ownership.
	// Returns ErrPhotoNotFound if the photo does not exist or is not pending.
	// Returns ErrPhotoNotOwned if the photo belongs to a different API key.
	FindPendingByIDAndAPIKey(ctx context.Context, id vo.PhotoID, apiKeyID string) (*entity.Photo, error)
}

// BrowseFilter contains filter options for browsing photos.
type BrowseFilter struct {
	// RouteID filters photos by route ID.
	RouteID string

	// STAStart filters photos with STA value >= this value.
	STAStart *float64

	// STAEnd filters photos with STA value <= this value.
	STAEnd *float64

	// Lane filters photos by lane code.
	Lane *string

	// Page is the page number (1-indexed).
	Page int

	// PerPage is the number of items per page.
	PerPage int
}

// SearchFilter contains filter options for searching photos.
type SearchFilter struct {
	// RouteIDs filters photos by multiple route IDs.
	RouteIDs []string

	// STARanges filters photos by STA value ranges.
	STARanges []STARange

	// Lanes filters photos by lane codes.
	Lanes []string

	// DateStart filters photos uploaded on or after this date.
	DateStart *time.Time

	// DateEnd filters photos uploaded on or before this date.
	DateEnd *time.Time

	// Tags filters photos that contain all specified tags.
	Tags []string

	// HasEXIFGPS filters photos that have or don't have GPS data.
	HasEXIFGPS *bool

	// Page is the page number (1-indexed).
	Page int

	// PerPage is the number of items per page.
	PerPage int
}

// STARange represents a range of STA values for filtering.
type STARange struct {
	// Start is the inclusive start of the STA range.
	Start float64

	// End is the inclusive end of the STA range.
	End float64
}

// BrowseResult contains the result of a browse or search operation.
type BrowseResult struct {
	// Photos is the list of photos matching the filter.
	Photos []*entity.Photo

	// TotalCount is the total number of photos matching the filter.
	TotalCount int64

	// Page is the current page number.
	Page int

	// PerPage is the number of items per page.
	PerPage int
}

// PendingUploadRepository manages upload tokens for two-phase upload process.
type PendingUploadRepository interface {
	// Create inserts a new pending upload into the repository.
	Create(ctx context.Context, upload *PendingUpload) error

	// GetByToken retrieves a pending upload by its token.
	GetByToken(ctx context.Context, token vo.UploadToken) (*PendingUpload, error)

	// GetByPhotoID retrieves a pending upload by the associated photo ID.
	GetByPhotoID(ctx context.Context, photoID vo.PhotoID) (*PendingUpload, error)

	// MarkAsCompleted marks the upload as fully completed (processed).
	MarkAsCompleted(ctx context.Context, token vo.UploadToken) error

	// MarkAsExpired marks uploads that have expired before the given time.
	// Returns the number of uploads marked as expired.
	MarkAsExpired(ctx context.Context, before time.Time) (int64, error)

	// Delete removes a pending upload from the repository.
	Delete(ctx context.Context, token vo.UploadToken) error

	// DeleteExpired removes all expired uploads before the given time.
	// Returns the number of uploads deleted.
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)

	// CountActiveByAPIKey returns the number of active (non-expired, non-completed) uploads for an API key.
	CountActiveByAPIKey(ctx context.Context, apiKeyID string) (int64, error)

	// GetExpired retrieves all expired uploads before the given time.
	GetExpired(ctx context.Context, before time.Time) ([]*PendingUpload, error)

	// ExpireTokensByPhotoID marks all pending tokens for a photo as expired.
	// This is used by the retry endpoint to invalidate existing tokens before generating new ones.
	ExpireTokensByPhotoID(ctx context.Context, photoID vo.PhotoID) error
}

// PendingUpload represents a pending upload record in the database.
type PendingUpload struct {
	// UploadToken is the unique token for this upload.
	UploadToken vo.UploadToken

	// PhotoID is the associated photo ID.
	PhotoID vo.PhotoID

	// APIKeyID is the API key ID that initiated the upload.
	APIKeyID string

	// CreatedAt is when the upload token was created.
	CreatedAt time.Time

	// ExpiresAt is when the upload token expires.
	ExpiresAt time.Time

	// Status is the current upload status.
	Status vo.UploadStatus
}

// APIKeyRepository manages API authentication keys.
type APIKeyRepository interface {
	// Create inserts a new API key into the repository.
	Create(ctx context.Context, apiKey *APIKey) error

	// GetByKeyHash retrieves an API key by its hashed value.
	GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)

	// GetByID retrieves an API key by its ID.
	GetByID(ctx context.Context, keyID string) (*APIKey, error)

	// UpdateLastUsed updates the last used timestamp for an API key.
	UpdateLastUsed(ctx context.Context, keyID string) error

	// Revoke revokes an API key.
	Revoke(ctx context.Context, keyID string) error

	// List retrieves all API keys, optionally filtering for active only.
	List(ctx context.Context, activeOnly bool) ([]*APIKey, error)

	// Delete permanently removes an API key from the repository.
	Delete(ctx context.Context, keyID string) error
}

// APIKey represents an API key record in the database.
type APIKey struct {
	// KeyID is the unique identifier for the API key.
	KeyID string

	// KeyHash is the SHA-256 hash of the actual API key.
	KeyHash string

	// Scopes defines the permissions granted by this API key (read, write, admin).
	Scopes []string

	// Description is a human-readable description of the API key.
	Description string

	// CreatedAt is when the API key was created.
	CreatedAt time.Time

	// ExpiresAt is when the API key expires (nil = never expires).
	ExpiresAt *time.Time

	// LastUsedAt is when the API key was last used.
	LastUsedAt *time.Time

	// IsActive indicates whether the API key is active.
	IsActive bool

	// RawKey is the unhashed API key (only used in tests, not stored in DB).
	RawKey string
}

// AuditLogRepository tracks photo operations for auditing purposes.
type AuditLogRepository interface {
	// Create inserts a new audit log entry.
	Create(ctx context.Context, entry *AuditLogEntry) error

	// GetByPhotoID retrieves audit log entries for a specific photo.
	GetByPhotoID(ctx context.Context, photoID vo.PhotoID, page, perPage int) ([]*AuditLogEntry, error)

	// GetByAPIKey retrieves audit log entries for a specific API key.
	GetByAPIKey(ctx context.Context, apiKeyID string, page, perPage int) ([]*AuditLogEntry, error)
}

// AuditLogEntry represents an audit log record in the database.
type AuditLogEntry struct {
	// LogID is the unique identifier for this log entry.
	LogID string

	// PhotoID is the associated photo ID (nil for operations without a photo).
	PhotoID *vo.PhotoID

	// Operation is the type of operation performed.
	Operation string

	// APIKeyID is the API key ID that performed the operation.
	APIKeyID string

	// OperatedAt is when the operation was performed.
	OperatedAt time.Time

	// Details contains additional operation-specific details.
	Details map[string]interface{}
}
