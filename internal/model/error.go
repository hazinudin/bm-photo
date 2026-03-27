package model

import (
	"errors"
	"fmt"
)

// Domain errors - Photo
var (
	ErrPhotoNotFound       = errors.New("photo not found")
	ErrPhotoAlreadyDeleted = errors.New("photo already deleted")
	ErrPhotoProcessing     = errors.New("photo is still processing")
	ErrPhotoNotReady       = errors.New("photo is not ready for this operation")
)

// Domain errors - Upload
var (
	ErrUploadTokenNotFound     = errors.New("upload token not found")
	ErrUploadTokenExpired      = errors.New("upload token has expired")
	ErrUploadTokenAlreadyUsed  = errors.New("upload token has already been used")
	ErrUploadTokenInvalidState = errors.New("upload token is in invalid state")
	ErrUploadInProgress        = errors.New("upload is already in progress")
)

// Domain errors - File
var (
	ErrFileTooLarge      = errors.New("file size exceeds maximum limit")
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrFileNotFound      = errors.New("file not found in storage")
)

// Domain errors - Validation
var (
	ErrInvalidRouteID     = errors.New("route_id is required")
	ErrInvalidLaneCode    = errors.New("lane_code must be in format L1-L10 or R1-R10")
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidSTAValue    = errors.New("sta_value must be >= 0")
)

// Domain errors - External
var (
	ErrRouteNotFound  = errors.New("route not found in LRS")
	ErrLRSUnavailable = errors.New("LRS service unavailable")
	ErrStorageError   = errors.New("storage operation failed")
)

// ValidationError represents a structured field-level validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error returns the error message in format "field: message".
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError with the given field and message.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// IsValidationError checks if error is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// GetValidationError extracts ValidationError if present.
func GetValidationError(err error) *ValidationError {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return ve
	}
	return nil
}
