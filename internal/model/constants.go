package model

import (
	"regexp"
	"time"
)

// LaneCodeRegex validates lane code format (L1-L10 or R1-R10)
var LaneCodeRegex = regexp.MustCompile(`^[LR]([1-9]|10)$`)

// File constraints
const (
	// MaxFileSizeBytes is the maximum allowed file size (10MB)
	MaxFileSizeBytes = 10 * 1024 * 1024

	// AllowedContentTypes is the comma-separated list of allowed content types
	AllowedContentTypes = "image/jpeg,image/png"
)

// Upload token constraints
const (
	// UploadTokenExpiryMinutes is the number of minutes before an upload token expires
	UploadTokenExpiryMinutes = 15

	// MaxPendingUploadsPerKey is the maximum number of pending uploads per API key
	MaxPendingUploadsPerKey = 10

	// MaxRetriesPerPhoto is the maximum number of upload retry attempts per photo
	MaxRetriesPerPhoto = 5
)

// Pagination defaults
const (
	// DefaultPage is the default page number for pagination
	DefaultPage = 1

	// DefaultPerPage is the default number of items per page
	DefaultPerPage = 20

	// MaxPerPage is the maximum number of items per page
	MaxPerPage = 100
)

// Rate limits (requests per minute per API key)
const (
	// RateLimitSignedURL is the rate limit for signed URL requests
	RateLimitSignedURL = 10

	// RateLimitComplete is the rate limit for complete upload requests
	RateLimitComplete = 10

	// RateLimitBrowse is the rate limit for browse requests
	RateLimitBrowse = 100

	// RateLimitTotal is the total rate limit per API key
	RateLimitTotal = 100
)

// STA constraints
const (
	// MinSTAValue is the minimum valid STA value
	MinSTAValue = 0.0
)

// UploadTokenExpiry is the duration before an upload token expires
var UploadTokenExpiry = 15 * time.Minute

// AllowedFileFormats is the list of allowed file formats
var AllowedFileFormats = []string{"JPEG", "PNG"}

// ValidUploadStatuses is the list of valid upload statuses
var ValidUploadStatuses = []string{"pending", "completed", "expired"}

// Cleanup defaults
const (
	// DefaultCleanupInterval is the default interval between cleanup runs
	DefaultCleanupInterval = 5 * time.Minute

	// DefaultCleanupRetention is the default retention period for expired uploads
	DefaultCleanupRetention = 24 * time.Hour
)
