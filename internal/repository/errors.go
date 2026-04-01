package repository

import "errors"

var (
	// ErrPhotoNotFound is returned when a photo cannot be found.
	ErrPhotoNotFound = errors.New("photo not found")

	// ErrDuplicatePhotoID is returned when attempting to create a photo with an ID that already exists.
	ErrDuplicatePhotoID = errors.New("photo ID already exists")

	// ErrTokenNotFound is returned when an upload token cannot be found.
	ErrTokenNotFound = errors.New("upload token not found")

	// ErrTokenExpired is returned when an upload token has expired.
	ErrTokenExpired = errors.New("upload token expired")

	// ErrTokenAlreadyUsed is returned when an upload token has already been used.
	ErrTokenAlreadyUsed = errors.New("upload token already used")

	// ErrRouteNotFound is returned when a route cannot be found.
	ErrRouteNotFound = errors.New("route not found")

	// ErrInvalidAPIKey is returned when an API key is invalid.
	ErrInvalidAPIKey = errors.New("invalid API key")

	// ErrAPIKeyNotFound is returned when an API key cannot be found.
	ErrAPIKeyNotFound = errors.New("api key not found")

	// ErrAPIKeyExpired is returned when an API key has expired.
	ErrAPIKeyExpired = errors.New("api key expired")

	// ErrAPIKeyRevoked is returned when an API key has been revoked.
	ErrAPIKeyRevoked = errors.New("api key revoked")

	// ErrAuditLogNotFound is returned when an audit log entry cannot be found.
	ErrAuditLogNotFound = errors.New("audit log not found")

	// ErrUploadNotFound is returned when an upload record cannot be found.
	ErrUploadNotFound = errors.New("upload not found")

	// ErrRetryLimitExceeded is returned when the maximum retry attempts have been exceeded.
	ErrRetryLimitExceeded = errors.New("maximum retry attempts exceeded")

	// ErrPhotoNotPending is returned when a photo is not in pending upload status.
	ErrPhotoNotPending = errors.New("photo is not in pending status")

	// ErrPhotoNotOwned is returned when a photo belongs to a different API key.
	ErrPhotoNotOwned = errors.New("photo is not owned by this API key")
)
