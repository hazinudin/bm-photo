package service

import (
	"errors"
)

// Service errors - distinct from domain/repository errors
// These represent service-layer specific failures

var (
	// ErrUploadQuotaExceeded indicates the API key has too many pending uploads
	ErrUploadQuotaExceeded = errors.New("upload quota exceeded for this API key")

	// ErrInvalidToken indicates the upload token is invalid
	ErrInvalidToken = errors.New("invalid upload token")

	// ErrTokenExpired indicates the upload token has expired
	ErrTokenExpired = errors.New("upload token has expired")

	// ErrTokenAlreadyUsed indicates the upload token has already been used
	ErrTokenAlreadyUsed = errors.New("upload token has already been used")

	// ErrTokenNotFound indicates the upload token was not found
	ErrTokenNotFound = errors.New("upload token not found")

	// ErrFileNotFound indicates the file was not found in GCS
	ErrFileNotFound = errors.New("file not found in storage")

	// ErrStorageError indicates a storage operation failed
	ErrStorageError = errors.New("storage operation failed")

	// ErrAPIKeyInvalid indicates the API key is invalid
	ErrAPIKeyInvalid = errors.New("invalid API key")

	// ErrAPIKeyInactive indicates the API key is inactive
	ErrAPIKeyInactive = errors.New("API key is inactive")

	// ErrAPIKeyExpired indicates the API key has expired
	ErrAPIKeyExpired = errors.New("API key has expired")

	// ErrScopeNotFound indicates the required scope is not present
	ErrScopeNotFound = errors.New("required scope not found")

	// ErrScopeRead indicates missing read scope
	ErrScopeRead = errors.New("read scope required")

	// ErrScopeWrite indicates missing write scope
	ErrScopeWrite = errors.New("write scope required")

	// ErrScopeAdmin indicates missing admin scope
	ErrScopeAdmin = errors.New("admin scope required")

	// ErrInvalidCoordinates indicates coordinates are invalid
	ErrInvalidCoordinates = errors.New("invalid coordinates")

	// ErrInvalidRouteID indicates route ID is invalid
	ErrInvalidRouteID = errors.New("invalid route ID format")

	// ErrInvalidLaneCode indicates lane code is invalid
	ErrInvalidLaneCode = errors.New("invalid lane code format")

	// ErrInvalidSTAValue indicates STA value is invalid
	ErrInvalidSTAValue = errors.New("invalid STA value")

	// ErrFileTooLarge indicates file exceeds size limit
	ErrFileTooLarge = errors.New("file size exceeds maximum limit")

	// ErrUnsupportedFormat indicates file format is not supported
	ErrUnsupportedFormat = errors.New("unsupported file format")

	// ErrPhotoNotFound indicates photo was not found
	ErrPhotoNotFound = errors.New("photo not found")

	// ErrPhotoDeleted indicates photo has been deleted
	ErrPhotoDeleted = errors.New("photo has been deleted")

	// ErrPhotoNotOwned indicates photo belongs to a different API key
	ErrPhotoNotOwned = errors.New("photo was created by a different API key")

	// ErrRetryLimitExceeded indicates the retry limit has been exceeded
	ErrRetryLimitExceeded = errors.New("maximum retry attempts exceeded")

	// ErrUploadNotPending indicates the upload is not in pending status
	ErrUploadNotPending = errors.New("upload is not in pending status")

	// ErrUnauthorized indicates the request is unauthorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the request is forbidden
	ErrForbidden = errors.New("forbidden")

	// ErrAPIKeyCreationFailed indicates API key creation failed
	ErrAPIKeyCreationFailed = errors.New("failed to create API key")

	// ErrAPIKeyNotActive indicates the API key is not active
	ErrAPIKeyNotActive = errors.New("API key is not active")

	// ErrAPIKeyNotFound indicates the API key was not found
	ErrAPIKeyNotFound = errors.New("API key not found")

	// ErrInvalidScope indicates an invalid scope was provided
	ErrInvalidScope = errors.New("invalid scope")
)

// ServiceError represents a service-layer error with additional context
type ServiceError struct {
	Code    string
	Message string
	Err     error
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// NewServiceError creates a new service error with context
func NewServiceError(code, message string, err error) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsServiceError checks if an error is a ServiceError
func IsServiceError(err error) bool {
	var se *ServiceError
	return errors.As(err, &se)
}

// GetServiceError extracts ServiceError if present
func GetServiceError(err error) *ServiceError {
	var se *ServiceError
	if errors.As(err, &se) {
		return se
	}
	return nil
}
