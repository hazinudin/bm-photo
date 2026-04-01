# Domain Layer Development Plan

**Version:** 1.3  
**Date:** April 1, 2026  
**Status:** ✅ Implementation Complete (Updated for Retry Endpoint Support)

---

## Overview

This document outlines the development plan for the domain layer (models) of the Bina Marga Survey Photo Service, following Domain-Driven Design (DDD) principles.

**Implementation Status:** ✅ Complete - All phases implemented as of March 27, 2026

**Updated:** Simplified architecture for MVP - removed processing status tracking, Phase 1 accepts all attributes, Phase 2 is confirmation only.

### Key Design Decisions

1. **Strong Types for IDs** - Use value objects for PhotoID, UploadToken instead of plain strings
2. **Rich Domain Models** - Entities contain validation logic and enforce invariants
3. **Struct Tags** - Use `db` and `json` tags for database/JSON mapping
4. **Aggregate Root Pattern** - Photo is the aggregate root; PendingUpload tracks upload workflow
5. **Separation of Concerns** - Domain models focus on business logic; auth/infrastructure concerns are separate
6. **Simplified STA** - STA confidence removed for MVP (can be added later if LRS provides it)
7. **No Processing Status** - Deferred processing (EXIF, thumbnails, LRS) removed for MVP simplicity

---

## Architecture

### Domain Layer Structure

```
internal/model/
├── entity/                    # Domain Entities (Aggregate Roots)
│   └── photo.go              # Photo aggregate root (main entity)
├── vo/                       # Value Objects (Strong Types)
│   ├── photo_id.go           # Photo UUID type
│   ├── upload_token.go       # Upload token type
│   ├── sta_source.go         # STA source enum
│   ├── file_format.go        # File format enum
│   ├── upload_status.go      # Upload status enum (pending/completed/expired)
│   └── coordinates.go        # Latitude/Longitude value object
├── dto/                      # Data Transfer Objects
│   ├── rest/                 # REST API DTOs
│   │   ├── upload.go         # Upload request/response DTOs
│   │   ├── photo.go          # Photo DTOs
│   │   └── browse.go         # Browse/search DTOs
│   └── errors.go             # Error response DTO
├── error.go                  # Domain errors
└── constants.go              # Domain constants

internal/auth/                 # Authentication (NOT domain model)
├── api_key.go                # API Key entity
└── api_key_repository.go     # API Key repository interface

proto/
└── photov1/
    └── photo.proto           # gRPC service definitions
```

---

## Phase 1: Value Objects

Value objects are immutable types that represent domain concepts with no identity.

### 1.1 PhotoID (`internal/model/vo/photo_id.go`)

Strong type for photo identification (UUID based).

```go
package vo

import (
    "errors"
    "github.com/google/uuid"
)

type PhotoID string

var (
    ErrInvalidPhotoID = errors.New("invalid photo ID format")
)

// NewPhotoID generates a new PhotoID with a random UUID
func NewPhotoID() PhotoID {
    return PhotoID(uuid.New().String())
}

// ParsePhotoID validates and creates a PhotoID from string
func ParsePhotoID(s string) (PhotoID, error) {
    if s == "" {
        return "", ErrInvalidPhotoID
    }
    if _, err := uuid.Parse(s); err != nil {
        return "", ErrInvalidPhotoID
    }
    return PhotoID(s), nil
}

// MustParsePhotoID panics on invalid input (use only in tests)
func MustParsePhotoID(s string) PhotoID {
    id, err := ParsePhotoID(s)
    if err != nil {
        panic(err)
    }
    return id
}

func (id PhotoID) String() string {
    return string(id)
}

func (id PhotoID) IsValid() bool {
    _, err := uuid.Parse(string(id))
    return err == nil
}

func (id PhotoID) IsZero() bool {
    return id == ""
}
```

### 1.2 UploadToken (`internal/model/vo/upload_token.go`)

Strong type for upload tracking token (UUID based).

```go
package vo

import (
    "errors"
    "github.com/google/uuid"
)

type UploadToken string

var (
    ErrInvalidUploadToken = errors.New("invalid upload token format")
)

func NewUploadToken() UploadToken {
    return UploadToken(uuid.New().String())
}

func ParseUploadToken(s string) (UploadToken, error) {
    if s == "" {
        return "", ErrInvalidUploadToken
    }
    if _, err := uuid.Parse(s); err != nil {
        return "", ErrInvalidUploadToken
    }
    return UploadToken(s), nil
}

func MustParseUploadToken(s string) UploadToken {
    token, err := ParseUploadToken(s)
    if err != nil {
        panic(err)
    }
    return token
}

func (t UploadToken) String() string {
    return string(t)
}

func (t UploadToken) IsValid() bool {
    _, err := uuid.Parse(string(t))
    return err == nil
}
```

### 1.3 STASource (`internal/model/vo/sta_source.go`)

Enum for Station value source.

```go
package vo

import (
    "errors"
    "strings"
)

type STASource string

const (
    STASourceUserProvided    STASource = "user_provided"
    STASourceLRSInterpolated STASource = "lrs_interpolated"
)

var (
    ErrInvalidSTASource = errors.New("invalid STA source")
)

func ParseSTASource(s string) (STASource, error) {
    source := STASource(strings.ToLower(s))
    switch source {
    case STASourceUserProvided, STASourceLRSInterpolated:
        return source, nil
    default:
        return "", ErrInvalidSTASource
    }
}

func (s STASource) String() string {
    return string(s)
}

func (s STASource) IsValid() bool {
    return s == STASourceUserProvided || s == STASourceLRSInterpolated
}
```

### 1.4 FileFormat (`internal/model/vo/file_format.go`)

Enum for supported file formats.

```go
package vo

import (
    "errors"
    "strings"
)

type FileFormat string

const (
    FileFormatJPEG FileFormat = "JPEG"
    FileFormatPNG  FileFormat = "PNG"
)

var (
    ErrInvalidFileFormat = errors.New("invalid file format")
)

func ParseFileFormat(s string) (FileFormat, error) {
    format := FileFormat(strings.ToUpper(s))
    switch format {
    case FileFormatJPEG, FileFormatPNG:
        return format, nil
    default:
        return "", ErrInvalidFileFormat
    }
}

func ParseFileFormatFromContentType(contentType string) (FileFormat, error) {
    switch strings.ToLower(contentType) {
    case "image/jpeg", "image/jpg":
        return FileFormatJPEG, nil
    case "image/png":
        return FileFormatPNG, nil
    default:
        return "", ErrInvalidFileFormat
    }
}

func (f FileFormat) String() string {
    return string(f)
}

func (f FileFormat) ContentType() string {
    switch f {
    case FileFormatJPEG:
        return "image/jpeg"
    case FileFormatPNG:
        return "image/png"
    default:
        return ""
    }
}

func (f FileFormat) IsValid() bool {
    return f == FileFormatJPEG || f == FileFormatPNG
}
```

### 1.5 UploadStatus (`internal/model/vo/upload_status.go`)

Enum for upload token status (tracked in pending_uploads table).

```go
package vo

import "errors"

type UploadStatus string

const (
    UploadStatusPending   UploadStatus = "pending"
    UploadStatusCompleted UploadStatus = "completed"
    UploadStatusExpired   UploadStatus = "expired"
)

var ErrInvalidUploadStatus = errors.New("invalid upload status")

func ParseUploadStatus(s string) (UploadStatus, error) {
    status := UploadStatus(s)
    if !status.IsValid() {
        return "", ErrInvalidUploadStatus
    }
    return status, nil
}

func (s UploadStatus) String() string {
    return string(s)
}

func (s UploadStatus) IsValid() bool {
    switch s {
    case UploadStatusPending, UploadStatusCompleted, UploadStatusExpired:
        return true
    default:
        return false
    }
}
```

**Note:** PhotoStatus enum has been removed. The MVP does not track processing status (EXIF extraction, thumbnail generation, LRS integration are deferred to future phases).

### 1.7 Coordinates (`internal/model/vo/coordinates.go`)

Value object for GPS coordinates.

```go
package vo

import (
    "errors"
    "math"
)

var (
    ErrInvalidLatitude  = errors.New("latitude must be between -90 and 90")
    ErrInvalidLongitude = errors.New("longitude must be between -180 and 180")
)

type Coordinates struct {
    latitude  float64
    longitude float64
}

func NewCoordinates(lat, lon float64) (Coordinates, error) {
    if lat < -90 || lat > 90 {
        return Coordinates{}, ErrInvalidLatitude
    }
    if lon < -180 || lon > 180 {
        return Coordinates{}, ErrInvalidLongitude
    }
    return Coordinates{
        latitude:  lat,
        longitude: lon,
    }, nil
}

func (c Coordinates) Latitude() float64 {
    return c.latitude
}

func (c Coordinates) Longitude() float64 {
    return c.longitude
}

func (c Coordinates) IsZero() bool {
    return c.latitude == 0 && c.longitude == 0
}
```

---

## Phase 2: Domain Entities

### 2.1 Photo Aggregate Root (`internal/model/entity/photo.go`)

The Photo entity is the aggregate root. It represents a survey photo with all its metadata.

**Important:** Processing status tracking (EXIF extraction, thumbnails, LRS) is deferred to future phases. The Photo entity focuses on storing photo metadata.

```go
package entity

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bina-marga/survey-photo/internal/model/vo"
)

// PhotoUploadStatus represents the upload lifecycle status
type PhotoUploadStatus string

const (
    PhotoUploadStatusPending   PhotoUploadStatus = "pending"
    PhotoUploadStatusCompleted PhotoUploadStatus = "completed"
    PhotoUploadStatusExpired   PhotoUploadStatus = "expired"
)

func (s PhotoUploadStatus) IsValid() bool {
    switch s {
    case PhotoUploadStatusPending, PhotoUploadStatusCompleted, PhotoUploadStatusExpired:
        return true
    default:
        return false
    }
}

// Photo represents a survey photo in the catalog.
// It is the aggregate root and stores all photo metadata.
type Photo struct {
    // Identity
    id vo.PhotoID

    // Location attributes
    routeID    string
    laneCode   string
    coordinates vo.Coordinates

    // Linear Reference System (optional - can be filled by other services)
    staValue  *float64
    staSource *vo.STASource

    // Storage paths
    gcsObjectName string

    // File metadata
    fileFormat       vo.FileFormat
    fileSizeBytes    int64
    originalFilename *string

    // User-provided metadata
    description *string
    tags        []string

    // Upload tracking (new fields for retry support)
    uploadStatus PhotoUploadStatus
    retryCount   int

    // Upload metadata
    uploadToken vo.UploadToken
    uploadedBy  string // API Key ID (string, references auth)
    uploadedAt  time.Time

    // Timestamps
    createdAt time.Time
    updatedAt time.Time

    // Soft delete
    deletedAt *time.Time
    deletedBy *string // API Key ID
}

// PhotoParams contains parameters for creating a new Photo
type PhotoParams struct {
	RouteID        string
	LaneCode       string
	Latitude       float64
	Longitude      float64
	STAValue       *float64 // Optional - can be nil
	STASource      *vo.STASource // Optional - can be nil

	FileFormat       vo.FileFormat
	FileSizeBytes    int64
	OriginalFilename *string

	UploadToken   vo.UploadToken
	UploadedBy    string // API Key ID
	GCSObjectName string
}

var (
	ErrInvalidRouteID          = errors.New("route_id is required")
	ErrInvalidLaneCode         = errors.New("lane_code must be in format L1-L10 or R1-R10")
	ErrInvalidFileSize         = errors.New("file_size_bytes must be greater than 0")
	ErrPhotoDeleted            = errors.New("photo has been deleted")
	ErrInvalidAPIKeyID         = errors.New("invalid API key ID")
	ErrPhotoNotFound           = errors.New("photo not found")
	ErrPhotoAlreadyCompleted   = errors.New("photo has already been uploaded and confirmed")
	ErrPhotoNotOwned           = errors.New("photo was created by a different API key")
	ErrRetryLimitExceeded      = errors.New("maximum retry attempts exceeded")
	ErrPhotoNotPending         = errors.New("photo is not in pending status")
)

// NewPhoto creates a new Photo entity with validation
func NewPhoto(params PhotoParams) (*Photo, error) {
	// Validate required fields
	if params.RouteID == "" {
		return nil, ErrInvalidRouteID
	}
	if err := validateLaneCode(params.LaneCode); err != nil {
		return nil, ErrInvalidLaneCode
	}
	if params.FileSizeBytes <= 0 {
		return nil, ErrInvalidFileSize
	}
	if params.UploadedBy == "" {
		return nil, ErrInvalidAPIKeyID
	}
	if !params.FileFormat.IsValid() {
		return nil, vo.ErrInvalidFileFormat
	}

	// Validate coordinates
	coords, err := vo.NewCoordinates(params.Latitude, params.Longitude)
	if err != nil {
		return nil, err
	}

	// Validate upload token
	if !params.UploadToken.IsValid() {
		return nil, vo.ErrInvalidUploadToken
	}

	now := time.Now()

	photo := &Photo{
		id:               vo.NewPhotoID(),
		routeID:          params.RouteID,
		laneCode:         params.LaneCode,
		coordinates:      coords,
		staValue:         params.STAValue,
		staSource:        params.STAValue,
		fileFormat:       params.FileFormat,
		fileSizeBytes:    params.FileSizeBytes,
		originalFilename: params.OriginalFilename,
		uploadToken:      params.UploadToken,
		uploadedBy:       params.UploadedBy,
		uploadedAt:       now,
		createdAt:        now,
		updatedAt:        now,
		tags:             []string{},
		gcsObjectName:    params.GCSObjectName,
	}

	return photo, nil
}

// validateLaneCode checks if lane code is in format L1-L10 or R1-R10
func validateLaneCode(code string) error {
    if code == "" {
        return errors.New("lane_code is required")
    }
    // Format: L1-L10 or R1-R10 (e.g., L1, L2, L10, R1, R3, R10)
    matched, _ := regexp.MatchString(`^[LR]\d{1,2}$`, code)
    if !matched {
        return errors.New("lane_code must be in format L1-L10 or R1-R10")
    }
    return nil
}

// Getters
func (p *Photo) ID() vo.PhotoID                    { return p.id }
func (p *Photo) RouteID() string                   { return p.routeID }
func (p *Photo) LaneCode() string                  { return p.laneCode }
func (p *Photo) Latitude() float64                  { return p.coordinates.Latitude() }
func (p *Photo) Longitude() float64                 { return p.coordinates.Longitude() }
func (p *Photo) STAValue() *float64                 { return p.staValue }
func (p *Photo) STASource() *vo.STASource          { return p.staSource }
func (p *Photo) GCSObjectName() string             { return p.gcsObjectName }
func (p *Photo) FileFormat() vo.FileFormat         { return p.fileFormat }
func (p *Photo) FileSizeBytes() int64              { return p.fileSizeBytes }
func (p *Photo) OriginalFilename() *string         { return p.originalFilename }
func (p *Photo) Description() *string              { return p.description }
func (p *Photo) Tags() []string                     { return p.tags }
func (p *Photo) UploadToken() vo.UploadToken       { return p.uploadToken }
func (p *Photo) UploadedBy() string                 { return p.uploadedBy }
func (p *Photo) UploadedAt() time.Time             { return p.uploadedAt }
func (p *Photo) CreatedAt() time.Time              { return p.createdAt }
func (p *Photo) UpdatedAt() time.Time               { return p.updatedAt }
func (p *Photo) DeletedAt() *time.Time              { return p.deletedAt }
func (p *Photo) DeletedBy() *string                 { return p.deletedBy }

// Upload tracking getters (new for retry support)
func (p *Photo) UploadStatus() PhotoUploadStatus  { return p.uploadStatus }
func (p *Photo) RetryCount() int                   { return p.retryCount }

// IsDeleted returns true if photo has been soft deleted
func (p *Photo) IsDeleted() bool {
    return p.deletedAt != nil
}

// IsUploadPending returns true if photo upload is pending
func (p *Photo) IsUploadPending() bool {
    return p.uploadStatus == PhotoUploadStatusPending
}

// IsUploadCompleted returns true if photo upload is completed
func (p *Photo) IsUploadCompleted() bool {
    return p.uploadStatus == PhotoUploadStatusCompleted
}

// CanRetryUpload checks if photo can be retried (pending status and retry count < 5)
func (p *Photo) CanRetryUpload() bool {
    return p.uploadStatus == PhotoUploadStatusPending && p.retryCount < 5
}

// HasSTA returns true if STA value has been set
func (p *Photo) HasSTA() bool {
    return p.staValue != nil
}

// Business Methods

// SetSTA sets the STA value and source (called by LRS integration later)
func (p *Photo) SetSTA(value float64, source vo.STASource) error {
    if value < 0 {
        return errors.New("sta_value must be >= 0")
    }
    if !source.IsValid() {
        return vo.ErrInvalidSTASource
    }
    p.staValue = &value
    p.staSource = &source
    p.updatedAt = time.Now()
    return nil
}

// UpdateDescription updates the photo description (only if not deleted)
func (p *Photo) UpdateDescription(desc string) error {
    if p.IsDeleted() {
        return ErrPhotoDeleted
    }
    p.description = &desc
    p.updatedAt = time.Now()
    return nil
}

// UpdateTags updates the photo tags (only if not deleted)
func (p *Photo) UpdateTags(tags []string) error {
    if p.IsDeleted() {
        return ErrPhotoDeleted
    }
    p.tags = tags
    p.updatedAt = time.Now()
    return nil
}

// UpdateLaneCode updates the lane code (only if not deleted)
func (p *Photo) UpdateLaneCode(laneCode string) error {
    if p.IsDeleted() {
        return ErrPhotoDeleted
    }
    if err := validateLaneCode(laneCode); err != nil {
        return ErrInvalidLaneCode
    }
    p.laneCode = laneCode
    p.updatedAt = time.Now()
    return nil
}

// SoftDelete marks the photo as deleted
func (p *Photo) SoftDelete(deletedByAPIKey string) error {
    if p.IsDeleted() {
        return ErrPhotoDeleted
    }
    now := time.Now()
    p.deletedAt = &now
    p.deletedBy = &deletedByAPIKey
    p.updatedAt = now
    return nil
}

// Restore un-deletes a soft-deleted photo (admin only)
func (p *Photo) Restore() error {
    if !p.IsDeleted() {
        return errors.New("photo is not deleted")
    }
    p.deletedAt = nil
    p.deletedBy = nil
    p.updatedAt = time.Now()
    return nil
}

// Upload Tracking Methods (new for retry support)

// MarkUploadCompleted marks the photo upload as completed
func (p *Photo) MarkUploadCompleted() error {
    if p.IsDeleted() {
        return ErrPhotoDeleted
    }
    if p.uploadStatus == PhotoUploadStatusCompleted {
        return ErrPhotoAlreadyCompleted
    }
    p.uploadStatus = PhotoUploadStatusCompleted
    p.updatedAt = time.Now()
    return nil
}

// MarkUploadExpired marks the photo upload as expired
func (p *Photo) MarkUploadExpired() error {
    if p.IsDeleted() {
        return ErrPhotoDeleted
    }
    if p.uploadStatus == PhotoUploadStatusCompleted {
        return ErrPhotoAlreadyCompleted
    }
    p.uploadStatus = PhotoUploadStatusExpired
    p.updatedAt = time.Now()
    return nil
}

// IncrementRetryCount increments the retry count for upload retries
func (p *Photo) IncrementRetryCount() error {
    if p.IsDeleted() {
        return ErrPhotoDeleted
    }
    if p.retryCount >= 5 {
        return ErrRetryLimitExceeded
    }
    p.retryCount++
    p.updatedAt = time.Now()
    return nil
}

// VerifyOwnership checks if the given API key is the owner of this photo
func (p *Photo) VerifyOwnership(apiKeyID string) error {
    if p.uploadedBy != apiKeyID {
        return ErrPhotoNotOwned
    }
    return nil
}
```

**Added for retry endpoint support:**
- `uploadStatus` field (PhotoUploadStatus: pending/completed/expired) for upload lifecycle tracking
- `retryCount` field (int, 0-5) for tracking upload retry attempts
- `UploadStatus()`, `RetryCount()`, `IsUploadPending()`, `IsUploadCompleted()`, `CanRetryUpload()` getter methods
- `MarkUploadCompleted()`, `MarkUploadExpired()`, `IncrementRetryCount()`, `VerifyOwnership()` business methods
- Photo upload status error types: `ErrPhotoNotFound`, `ErrPhotoAlreadyCompleted`, `ErrPhotoNotOwned`, `ErrRetryLimitExceeded`, `ErrPhotoNotPending`
- `GetNewSignedURLResponse` DTO for retry endpoint

**Removed from previous version:**
- `status` field (processing status tracking removed for MVP)
- `exifData` field (EXIF extraction deferred)
- `thumbnailSmallPath`, `thumbnailMediumPath`, `thumbnailLargePath` fields (thumbnails deferred)
- `processingCompletedAt` field
- `MarkProcessingComplete()`, `MarkProcessingFailed()` methods
- `IsReady()`, `IsProcessing()`, `IsFailed()` methods
- `GenerateGCSObjectName()`, `GenerateThumbnailPaths()`, `GenerateOriginalPath()` methods (GCS object name is now passed as a parameter)

---

## Phase 3: REST API DTOs

DTOs for HTTP request/response mapping. Separate from domain entities.

### 3.1 Upload DTOs (`internal/model/dto/rest/upload.go`)

```go
package rest

import (
    "errors"
    "regexp"
    "time"

    "github.com/bina-marga/survey-photo/internal/model/vo"
)

// GetSignedUploadURLRequest - Phase 1: Request signed upload URL with all attributes
type GetSignedUploadURLRequest struct {
    FileMetadata    FileMetadata    `json:"file_metadata"`
    PhotoAttributes PhotoAttributes `json:"photo_attributes"`
}

type FileMetadata struct {
    Filename      string `json:"filename"`
    ContentType   string `json:"content_type"`
    FileSizeBytes int64  `json:"file_size_bytes"`
}

type PhotoAttributes struct {
    RouteID     string   `json:"route_id"`
    LaneCode    string   `json:"lane_code"`
    Latitude    float64  `json:"latitude"`
    Longitude   float64  `json:"longitude"`
    STAValue    *float64 `json:"sta_value,omitempty"`
    Description *string `json:"description,omitempty"`
    Tags        []string `json:"tags,omitempty"`
}

func (r *GetSignedUploadURLRequest) Validate() error {
    // Validate file metadata
    if r.FileMetadata.Filename == "" {
        return &ValidationError{Field: "file_metadata.filename", Message: "filename is required"}
    }
    if r.FileMetadata.ContentType == "" {
        return &ValidationError{Field: "file_metadata.content_type", Message: "content_type is required"}
    }
    if !isValidContentType(r.FileMetadata.ContentType) {
        return &ValidationError{Field: "file_metadata.content_type", Message: "must be image/jpeg or image/png"}
    }
    if r.FileMetadata.FileSizeBytes <= 0 {
        return &ValidationError{Field: "file_metadata.file_size_bytes", Message: "file_size_bytes must be greater than 0"}
    }
    if r.FileMetadata.FileSizeBytes > MaxFileSizeBytes {
        return &ValidationError{Field: "file_metadata.file_size_bytes", Message: "file size exceeds maximum of 10MB"}
    }

    // Validate photo attributes
    if r.PhotoAttributes.RouteID == "" {
        return &ValidationError{Field: "photo_attributes.route_id", Message: "route_id is required"}
    }
    if err := validateLaneCode(r.PhotoAttributes.LaneCode); err != nil {
        return &ValidationError{Field: "photo_attributes.lane_code", Message: err.Error()}
    }
    if r.PhotoAttributes.Latitude < -90 || r.PhotoAttributes.Latitude > 90 {
        return &ValidationError{Field: "photo_attributes.latitude", Message: "latitude must be between -90 and 90"}
    }
    if r.PhotoAttributes.Longitude < -180 || r.PhotoAttributes.Longitude > 180 {
        return &ValidationError{Field: "photo_attributes.longitude", Message: "longitude must be between -180 and 180"}
    }
    if r.PhotoAttributes.STAValue != nil && *r.PhotoAttributes.STAValue < 0 {
        return &ValidationError{Field: "photo_attributes.sta_value", Message: "sta_value must be >= 0"}
    }
    return nil
}

func isValidContentType(ct string) bool {
    return ct == "image/jpeg" || ct == "image/png"
}

// validateLaneCode checks if lane code is in format L1-L10 or R1-R10
func validateLaneCode(code string) error {
    if code == "" {
        return errors.New("lane_code is required")
    }
    matched, _ := regexp.MatchString(`^[LR]\d{1,2}$`, code)
    if !matched {
        return errors.New("lane_code must be in format L1-L10 or R1-R10")
    }
    return nil
}

// GetSignedUploadURLResponse - Phase 1: Response with signed URL
type GetSignedUploadURLResponse struct {
    UploadToken vo.UploadToken `json:"upload_token"`
    SignedURL   string        `json:"signed_url"`
    PhotoID     vo.PhotoID    `json:"photo_id"`
    ExpiresAt   time.Time     `json:"expires_at"`
}

// GetNewSignedURLResponse - Retry Response with new signed URL for existing photo
type GetNewSignedURLResponse struct {
    PhotoID     vo.PhotoID `json:"photo_id"`
    UploadToken vo.UploadToken `json:"upload_token"`
    SignedURL   string     `json:"signed_url"`
    ExpiresAt   time.Time  `json:"expires_at"`
    RetryCount  int        `json:"retry_count"`
    MaxRetries  int        `json:"max_retries"`
}

// ConfirmUploadRequest - Phase 2: Confirm successful GCS upload
type ConfirmUploadRequest struct {
    UploadToken vo.UploadToken `json:"upload_token"`
}

func (r *ConfirmUploadRequest) Validate() error {
    if !r.UploadToken.IsValid() {
        return &ValidationError{Field: "upload_token", Message: "invalid upload token format"}
    }
    return nil
}

// ConfirmUploadResponse - Phase 2: Confirmation response
type ConfirmUploadResponse struct {
    PhotoID vo.PhotoID `json:"photo_id"`
    Message string    `json:"message"`
}
```

### 3.2 Photo DTOs (`internal/model/dto/rest/photo.go`)

```go
package rest

import (
    "time"

    "github.com/bina-marga/survey-photo/internal/model/vo"
)

// PhotoResponse - Full photo metadata response
type PhotoResponse struct {
    PhotoID       vo.PhotoID    `json:"photo_id"`
    RouteID       string        `json:"route_id"`
    LaneCode      string        `json:"lane_code"`
    Latitude      float64       `json:"latitude"`
    Longitude     float64       `json:"longitude"`
    STAValue      *float64      `json:"sta_value,omitempty"`
    STASource     *vo.STASource `json:"sta_source,omitempty"`
    FileFormat    vo.FileFormat `json:"file_format"`
    FileSizeBytes int64         `json:"file_size_bytes"`
    Description   *string       `json:"description,omitempty"`
    Tags          []string      `json:"tags"`
    UploadedAt    time.Time     `json:"uploaded_at"`
    DownloadURL   string        `json:"download_url"`
}

// UpdatePhotoRequest - Update photo metadata
type UpdatePhotoRequest struct {
    Description *string  `json:"description,omitempty"`
    Tags        []string `json:"tags,omitempty"`
    LaneCode    *string  `json:"lane_code,omitempty"`
}

func (r *UpdatePhotoRequest) Validate() error {
    if r.LaneCode != nil {
        if err := validateLaneCode(*r.LaneCode); err != nil {
            return &ValidationError{Field: "lane_code", Message: "lane_code must be in format L1-L10 or R1-R10"}
        }
    }
    return nil
}

// UpdatePhotoResponse - Confirmation of update
type UpdatePhotoResponse struct {
    PhotoID     vo.PhotoID `json:"photo_id"`
    Description *string    `json:"description,omitempty"`
    Tags        []string   `json:"tags"`
    LaneCode    string     `json:"lane_code"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

// DeletePhotoResponse - Confirmation of deletion
type DeletePhotoResponse struct {
    PhotoID      vo.PhotoID `json:"photo_id"`
    DeletedAt    time.Time  `json:"deleted_at"`
    DeletionType string     `json:"deletion_type"` // "soft" or "hard"
}
```

**Removed from previous version:**
- `EXIFData` field and `EXIFDataDTO` (EXIF extraction deferred)
- `ThumbnailURLs` struct (thumbnails deferred)
- `Status` field (processing status tracking removed for MVP)

### 3.3 Browse DTOs (`internal/model/dto/rest/browse.go`)

```go
package rest

import "github.com/bina-marga/survey-photo/internal/model/vo"

// BrowsePhotosRequest - Query photos with filters
type BrowsePhotosRequest struct {
    RouteID  string   `query:"route_id"`
    STAStart *float64  `query:"sta_start"`
    STAEnd   *float64  `query:"sta_end"`
    Lane     *int      `query:"lane"`
    Page     int       `query:"page"`
    PerPage  int       `query:"per_page"`
}

func (r *BrowsePhotosRequest) Validate() error {
    if r.RouteID == "" {
        return &ValidationError{Field: "route_id", Message: "route_id is required"}
    }
    if r.STAStart != nil && *r.STAStart < 0 {
        return &ValidationError{Field: "sta_start", Message: "sta_start must be >= 0"}
    }
    if r.STAEnd != nil && *r.STAEnd < 0 {
        return &ValidationError{Field: "sta_end", Message: "sta_end must be >= 0"}
    }
    if r.STAStart != nil && r.STAEnd != nil && *r.STAStart > *r.STAEnd {
        return &ValidationError{Field: "sta_start", Message: "sta_start must be <= sta_end"}
    }
    if r.Lane != nil && *r.Lane <= 0 {
        return &ValidationError{Field: "lane", Message: "lane must be greater than 0"}
    }
    if r.Page <= 0 {
        r.Page = DefaultPage
    }
    if r.PerPage <= 0 || r.PerPage > MaxPerPage {
        r.PerPage = DefaultPerPage
    }
    return nil
}

// BrowsePhotosResponse - Paginated photo list
type BrowsePhotosResponse struct {
    Photos     []PhotoSummary `json:"photos"`
    Pagination Pagination     `json:"pagination"`
}

type PhotoSummary struct {
    PhotoID      vo.PhotoID `json:"photo_id"`
    RouteID      string     `json:"route_id"`
    LaneCode     string     `json:"lane_code"`
    STAValue     float64    `json:"sta_value"`
    ThumbnailURL string     `json:"thumbnail_url"`
    UploadedAt   time.Time  `json:"uploaded_at"`
}

type Pagination struct {
    CurrentPage int   `json:"current_page"`
    PerPage     int   `json:"per_page"`
    TotalCount  int64 `json:"total_count"`
    TotalPages  int   `json:"total_pages"`
}

// SearchPhotosRequest - Advanced search with multiple filters
type SearchPhotosRequest struct {
    RouteIDs    []string  `json:"route_ids"`
    STARanges   []STARange `json:"sta_ranges"`
    Lanes       []int      `json:"lanes"`
    DateStart   *time.Time `json:"date_start"`
    DateEnd     *time.Time `json:"date_end"`
    Tags        []string   `json:"tags"`
    HasEXIFGPS  *bool      `json:"has_exif_gps"`
    Page        int        `json:"page"`
    PerPage     int        `json:"per_page"`
}

type STARange struct {
    Start float64 `json:"start"`
    End   float64 `json:"end"`
}

func (r *SearchPhotosRequest) Validate() error {
    if r.Page <= 0 {
        r.Page = DefaultPage
    }
    if r.PerPage <= 0 || r.PerPage > MaxPerPage {
        r.PerPage = DefaultPerPage
    }
    return nil
}

// SearchPhotosResponse - Paginated search results
type SearchPhotosResponse struct {
    Photos     []PhotoSummary `json:"photos"`
    Pagination Pagination      `json:"pagination"`
}
```

---

## Phase 4: Domain Errors

### `internal/model/error.go`

```go
package model

import (
    "errors"
    "fmt"
)

// Domain errors - Photo
var (
    ErrPhotoNotFound         = errors.New("photo not found")
    ErrPhotoAlreadyDeleted   = errors.New("photo already deleted")
    ErrPhotoAlreadyCompleted = errors.New("photo has already been uploaded and confirmed")
    ErrPhotoNotOwned         = errors.New("photo was created by a different API key")
    ErrPhotoNotPending       = errors.New("photo is not in pending status")
)

// Domain errors - Upload
var (
    ErrUploadTokenNotFound     = errors.New("upload token not found")
    ErrUploadTokenExpired      = errors.New("upload token has expired")
    ErrUploadTokenAlreadyUsed  = errors.New("upload token has already been used")
    ErrUploadTokenInvalidState = errors.New("upload token is in invalid state")
    ErrUploadInProgress        = errors.New("upload is already in progress")
    ErrRetryLimitExceeded      = errors.New("maximum retry attempts exceeded")
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
```

---

## Phase 5: Domain Constants

### `internal/model/constants.go`

```go
package model

import "time"

// File constraints
const (
    MaxFileSizeBytes = 10 * 1024 * 1024 // 10MB
    AllowedContentTypes = "image/jpeg,image/png"
)

// Upload token constraints
const (
    UploadTokenExpiryMinutes = 15
    MaxPendingUploadsPerKey  = 10
    MaxRetriesPerPhoto       = 5
)

// Pagination defaults
const (
    DefaultPage    = 1
    DefaultPerPage = 20
    MaxPerPage     = 100
)

// Rate limits (requests per minute per API key)
const (
    RateLimitSignedURL = 10
    RateLimitConfirm   = 10
    RateLimitBrowse    = 100
    RateLimitTotal     = 100
)

// STA constraints
const (
    MinSTAValue = 0.0
)

// Upload token expiry duration
var UploadTokenExpiry = 15 * time.Minute

// File formats
var AllowedFileFormats = []string{"JPEG", "PNG"}

// Upload statuses (pending_uploads table)
var ValidUploadStatuses = []string{"pending", "completed", "expired"}
```

**Note:** Photo status enum has been removed. Thumbnail constants removed (thumbnails deferred to future phase).

---

## Phase 6: gRPC Proto Definitions

### `proto/photov1/photo.proto`

```protobuf
syntax = "proto3";

package bina_marga.survey_photo.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

option go_package = "github.com/bina-marga/survey-photo/gen/go/photov1;photov1";

// Service for browsing photo catalog (internal microservices only)
// Upload operations use REST API
service PhotoCatalogService {
    // Get a single photo by ID
    rpc GetPhoto(GetPhotoRequest) returns (GetPhotoResponse);
    
    // Browse photos by route with optional STA and lane filters
    rpc BrowsePhotos(BrowsePhotosRequest) returns (BrowsePhotosResponse);
    
    // Advanced search with multiple filters
    rpc SearchPhotos(SearchPhotosRequest) returns (SearchPhotosResponse);
    
    // Get photo status (processing, ready, failed)
    rpc GetPhotoStatus(GetPhotoStatusRequest) returns (GetPhotoStatusResponse);
    
    // Batch get multiple photos by IDs
    rpc BatchGetPhotos(BatchGetPhotosRequest) returns (BatchGetPhotosResponse);
}

// Browse photos by route, STA range, and lane
message BrowsePhotosRequest {
    string route_id = 1;
    optional double sta_start = 2;
    optional double sta_end = 3;
    optional string lane_code = 4;
    int32 page = 5;
    int32 per_page = 6;
}

message BrowsePhotosResponse {
    repeated PhotoMetadata photos = 1;
    Pagination pagination = 2;
}

message Pagination {
    int32 current_page = 1;
    int32 per_page = 2;
    int64 total_count = 3;
    int32 total_pages = 4;
}

// Get single photo
message GetPhotoRequest {
    string photo_id = 1;
}

message GetPhotoResponse {
    PhotoMetadata photo = 1;
    string download_url = 2;
    map<string, string> thumbnail_urls = 3; // small, medium, large
}

// Advanced search
message SearchPhotosRequest {
    repeated string route_ids = 1;
    repeated STARange sta_ranges = 2;
    repeated int32 lanes = 3;
    optional google.protobuf.Timestamp date_start = 4;
    optional google.protobuf.Timestamp date_end = 5;
    repeated string tags = 6;
    int32 page = 7;
    int32 per_page = 8;
}

message STARange {
    double start = 1;
    double end = 2;
}

message SearchPhotosResponse {
    repeated PhotoMetadata photos = 1;
    Pagination pagination = 2;
}

// Photo status
message GetPhotoStatusRequest {
    string photo_id = 1;
}

message GetPhotoStatusResponse {
    string photo_id = 1;
    string status = 2; // processing, ready, failed
    bool thumbnails_ready = 3;
    bool exif_extracted = 4;
    bool sta_calculated = 5;
    optional string error_message = 6;
    optional google.protobuf.Timestamp processed_at = 7;
}

// Batch get
message BatchGetPhotosRequest {
    repeated string photo_ids = 1;
}

message BatchGetPhotosResponse {
    repeated PhotoMetadata photos = 1;
}

// Common message types
message PhotoMetadata {
    string photo_id = 1;
    string route_id = 2;
    string lane_code = 3;
    double latitude = 4;
    double longitude = 5;
    double sta_value = 6;
    string sta_source = 7;
    string file_format = 8;
    int64 file_size_bytes = 9;
    string description = 10;
    repeated string tags = 11;
    google.protobuf.Timestamp uploaded_at = 12;
    string status = 13;
}
```

---

## Implementation Order

| Week | Phase | Files Created | Status |
|------|-------|---------------|--------|
| 1 | Value Objects | `vo/photo_id.go`, `vo/upload_token.go`, `vo/sta_source.go`, `vo/file_format.go`, `vo/photo_status.go`, `vo/upload_status.go`, `vo/coordinates.go` | ✅ Complete |
| 1 | Constants & Errors | `constants.go`, `error.go` | ✅ Complete |
| 1 | Tests | `vo/*_test.go`, `error_test.go` | ✅ Complete |
| 2 | Core Entities | `entity/photo.go` | ✅ Complete |
| 2 | Entity Tests | `entity/*_test.go` | ✅ Complete |
| 3 | REST DTOs | `dto/rest/upload.go`, `dto/rest/photo.go`, `dto/rest/browse.go` | ✅ Complete |
| 3 | DTO Validation Tests | `dto/rest/*_test.go` | ✅ Complete |
| 4 | Proto | `proto/photov1/photo.proto` | ❌ Pending (see repository-layer-plan.md) |

---

## Test Strategy

### Unit Tests per File

Every value object, entity, and DTO must have corresponding test files.

**Test File Naming:**
- `vo/photo_id.go` → `vo/photo_id_test.go`
- `entity/photo.go` → `entity/photo_test.go`
- `dto/rest/upload.go` → `dto/rest/upload_test.go`

**Test Structure:**
```go
func TestPhotoID_NewPhotoID(t *testing.T) {
    id := vo.NewPhotoID()
    assert.True(t, id.IsValid())
    assert.NotEmpty(t, id.String())
}

func TestPhotoID_ParsePhotoID_Valid(t *testing.T) {
    validUUID := "550e8400-e29b-41d4-a716-446655440000"
    id, err := vo.ParsePhotoID(validUUID)
    assert.NoError(t, err)
    assert.Equal(t, validUUID, id.String())
}

func TestPhotoID_ParsePhotoID_Invalid(t *testing.T) {
    tests := []struct {
        name string
        input string
    }{
        {"empty", ""},
        {"invalid", "not-a-uuid"},
        {"wrong format", "550e8400e29b41d4a716446655440000"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := vo.ParsePhotoID(tt.input)
            assert.Error(t, err)
            assert.ErrorIs(t, err, vo.ErrInvalidPhotoID)
        })
    }
}
```

---

## Review Checklist

### Value Objects
- [ ] All IDs use strong types (PhotoID, UploadToken)
- [ ] Parse functions validate input
- [ ] IsValid methods check format
- [ ] String() methods return underlying string
- [ ] All VOs are immutable (no setters)

### Entities
- [ ] Constructor validates all required fields
- [ ] Business logic methods enforce invariants
- [ ] State transitions are protected (cannot skip states)
- [ ] Soft delete is implemented correctly
- [ ] Getters expose all fields
- [ ] Update methods return errors for invalid states

### DTOs
- [ ] JSON tags use snake_case
- [ ] Validate methods check all required fields
- [ ] Optional fields use pointers
- [ ] Pagination has defaults
- [ ] Error responses use ValidationError struct

### General
- [ ] No external dependencies in model package
- [ ] All errors are defined in error.go
- [ ] All constants in constants.go
- [ ] Import groups: stdlib, third-party, internal
- [ ] Files use snake_case.go naming
- [ ] All public types have documentation comments

---

## Next Steps

Domain layer implementation is complete. Proceed to:

1. **Repository Layer** - Implement database repositories per `repository-layer-plan.md`
2. **Service Layer** - Implement business logic orchestration
3. **Handler Layer** - Implement REST API handlers using domain models
4. **Infrastructure** - Database migrations, GCS client, LRS client, configuration

---

## References

- PRD: `/PRD.md`
- AGENTS.md: `/AGENTS.md`
- Database Schema: See PRD Section 5.2