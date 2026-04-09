package entity

import (
	"errors"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
)

// Photo represents a survey photo in the catalog.
// It is the aggregate root and manages its upload lifecycle.
type Photo struct {
	// Identity
	id vo.PhotoID

	// Location attributes
	routeID     string
	laneCode    string
	surveyYear  int
	coordinates *vo.Coordinates

	// Linear Reference System
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

	// Upload metadata
	uploadToken  vo.UploadToken
	uploadStatus vo.UploadStatus
	uploadedBy   string // API Key ID (string, references auth)
	uploadedAt   time.Time

	// Upload retry tracking
	retryCount int

	// Timestamps
	createdAt time.Time
	updatedAt time.Time

	// Soft delete
	deletedAt *time.Time
	deletedBy *string // API Key ID
}

// PhotoParams contains parameters for creating a new Photo.
type PhotoParams struct {
	RouteID    string
	LaneCode   string
	SurveyYear int

	GCSObjectName    string
	FileFormat       vo.FileFormat
	FileSizeBytes    int64
	OriginalFilename *string

	UploadToken vo.UploadToken
	UploadedBy  string // API Key ID
}

var (
	ErrInvalidRouteID  = errors.New("route_id is required")
	ErrInvalidLaneCode = errors.New("lane_code must be in format L1-L10 or R1-R10")
	ErrInvalidSTAValue = errors.New("sta_value must be greater than or equal to 0")
	ErrInvalidFileSize = errors.New("file_size_bytes must be greater than 0")
	ErrPhotoDeleted    = errors.New("photo has been deleted")
	ErrInvalidAPIKeyID = errors.New("invalid API key ID")

	ErrUploadNotPending      = errors.New("upload is not in pending state")
	ErrPhotoNotDeleted       = errors.New("photo is not deleted")
	ErrPhotoAlreadyCompleted = errors.New("photo has already been uploaded and confirmed")
	ErrPhotoNotOwned         = errors.New("photo was created by a different API key")
	ErrRetryLimitExceeded    = errors.New("maximum retry attempts exceeded")
	ErrPhotoNotPending       = errors.New("photo is not in pending status")
)

// NewPhoto creates a new Photo entity with validation.
func NewPhoto(params PhotoParams) (*Photo, error) {
	// Validate required fields
	if params.RouteID == "" {
		return nil, ErrInvalidRouteID
	}
	if params.GCSObjectName == "" {
		return nil, errors.New("gcs_object_name is required")
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

	// Validate upload token
	if !params.UploadToken.IsValid() {
		return nil, vo.ErrInvalidUploadToken
	}

	// Validate lane code if provided
	if params.LaneCode != "" && !IsValidLaneCode(params.LaneCode) {
		return nil, ErrInvalidLaneCode
	}

	now := time.Now()

	surveyYear := params.SurveyYear
	if surveyYear == 0 {
		surveyYear = now.Year()
	}

	photo := &Photo{
		id:               vo.NewPhotoID(),
		routeID:          params.RouteID,
		laneCode:         params.LaneCode,
		surveyYear:       surveyYear,
		gcsObjectName:    params.GCSObjectName,
		fileFormat:       params.FileFormat,
		fileSizeBytes:    params.FileSizeBytes,
		originalFilename: params.OriginalFilename,
		uploadToken:      params.UploadToken,
		uploadStatus:     vo.UploadStatusPending,
		uploadedBy:       params.UploadedBy,
		uploadedAt:       now,
		retryCount:       0,
		createdAt:        now,
		updatedAt:        now,
		tags:             []string{},
	}

	return photo, nil
}

// IsValidLaneCode validates that the lane code matches the format L1-L10 or R1-R10
func IsValidLaneCode(code string) bool {
	return model.LaneCodeRegex.MatchString(code)
}

// Getters

// ID returns the photo ID
func (p *Photo) ID() vo.PhotoID {
	return p.id
}

// RouteID returns the route ID
func (p *Photo) RouteID() string {
	return p.routeID
}

// LaneCode returns the lane code
func (p *Photo) LaneCode() string {
	return p.laneCode
}

// SurveyYear returns the survey year
func (p *Photo) SurveyYear() int {
	return p.surveyYear
}

// Latitude returns the latitude coordinate (0 if not set)
func (p *Photo) Latitude() *float64 {
	if p.coordinates == nil {
		return nil
	}
	v := p.coordinates.Latitude()
	return &v
}

// Longitude returns the longitude coordinate (0 if not set)
func (p *Photo) Longitude() *float64 {
	if p.coordinates == nil {
		return nil
	}
	v := p.coordinates.Longitude()
	return &v
}

// STAValue returns the STA value
func (p *Photo) STAValue() *float64 {
	return p.staValue
}

// STASource returns the STA source
func (p *Photo) STASource() *vo.STASource {
	return p.staSource
}

// GCSObjectName returns the GCS object name
func (p *Photo) GCSObjectName() string {
	return p.gcsObjectName
}

// FileFormat returns the file format
func (p *Photo) FileFormat() vo.FileFormat {
	return p.fileFormat
}

// FileSizeBytes returns the file size in bytes
func (p *Photo) FileSizeBytes() int64 {
	return p.fileSizeBytes
}

// OriginalFilename returns the original filename
func (p *Photo) OriginalFilename() *string {
	return p.originalFilename
}

// Description returns the photo description
func (p *Photo) Description() *string {
	return p.description
}

// Tags returns the photo tags
func (p *Photo) Tags() []string {
	return p.tags
}

// UploadToken returns the upload token
func (p *Photo) UploadToken() vo.UploadToken {
	return p.uploadToken
}

// UploadStatus returns the upload status
func (p *Photo) UploadStatus() vo.UploadStatus {
	return p.uploadStatus
}

// UploadedBy returns the API key ID that uploaded the photo
func (p *Photo) UploadedBy() string {
	return p.uploadedBy
}

// UploadedAt returns the upload timestamp
func (p *Photo) UploadedAt() time.Time {
	return p.uploadedAt
}

// CreatedAt returns the creation timestamp
func (p *Photo) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update timestamp
func (p *Photo) UpdatedAt() time.Time {
	return p.updatedAt
}

// DeletedAt returns the deletion timestamp
func (p *Photo) DeletedAt() *time.Time {
	return p.deletedAt
}

// DeletedBy returns the API key ID that deleted the photo
func (p *Photo) DeletedBy() *string {
	return p.deletedBy
}

// IsDeleted returns true if photo has been soft deleted
func (p *Photo) IsDeleted() bool {
	return p.deletedAt != nil
}

// IsUploadPending returns true if upload token is in pending state
func (p *Photo) IsUploadPending() bool {
	return p.uploadStatus == vo.UploadStatusPending
}

// IsUploadCompleted returns true if upload is completed
func (p *Photo) IsUploadCompleted() bool {
	return p.uploadStatus == vo.UploadStatusCompleted
}

// CanRetryUpload checks if the photo upload can be retried
func (p *Photo) CanRetryUpload() bool {
	return p.uploadStatus == vo.UploadStatusPending && p.retryCount < model.MaxRetriesPerPhoto
}

// RetryCount returns the current retry count
func (p *Photo) RetryCount() int {
	return p.retryCount
}

// Business Methods

// MarkUploadComplete transitions upload status to completed.
func (p *Photo) MarkUploadComplete() error {
	if p.uploadStatus != vo.UploadStatusPending {
		return ErrUploadNotPending
	}
	p.uploadStatus = vo.UploadStatusCompleted
	p.updatedAt = time.Now()
	return nil
}

// MarkUploadCompleted marks the photo upload as completed (alias for MarkUploadComplete).
func (p *Photo) MarkUploadCompleted() error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	if p.uploadStatus == vo.UploadStatusCompleted {
		return ErrPhotoAlreadyCompleted
	}
	if p.uploadStatus != vo.UploadStatusPending {
		return ErrPhotoNotPending
	}
	p.uploadStatus = vo.UploadStatusCompleted
	p.updatedAt = time.Now()
	return nil
}

// MarkUploadExpired marks the photo upload as expired.
func (p *Photo) MarkUploadExpired() error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	if p.uploadStatus == vo.UploadStatusCompleted {
		return ErrPhotoAlreadyCompleted
	}
	p.uploadStatus = vo.UploadStatusExpired
	p.updatedAt = time.Now()
	return nil
}

// IncrementRetryCount increments the retry count for upload retries.
func (p *Photo) IncrementRetryCount() error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	if p.retryCount >= model.MaxRetriesPerPhoto {
		return ErrRetryLimitExceeded
	}
	p.retryCount++
	p.updatedAt = time.Now()
	return nil
}

// VerifyOwnership checks if the given API key is the owner of this photo.
func (p *Photo) VerifyOwnership(apiKeyID string) error {
	if p.uploadedBy != apiKeyID {
		return ErrPhotoNotOwned
	}
	return nil
}

// SetSTA sets the STA value and source.
func (p *Photo) SetSTA(value float64, source vo.STASource) error {
	if value < 0 {
		return model.ErrInvalidSTAValue
	}
	if !source.IsValid() {
		return vo.ErrInvalidSTASource
	}
	p.staValue = &value
	p.staSource = &source
	p.updatedAt = time.Now()
	return nil
}

// Set lane code with validation.
func (p *Photo) SetLaneCode(code string) error {
	if !IsValidLaneCode(code) {
		return ErrInvalidLaneCode
	}
	p.laneCode = code
	p.updatedAt = time.Now()
	return nil
}

// Set coordinate with validation.
func (p *Photo) SetCoordinates(lat, lon float64) error {
	coords, err := vo.NewCoordinates(lat, lon)
	if err != nil {
		return err
	}
	p.coordinates = &coords
	p.updatedAt = time.Now()
	return nil
}

// UpdateDescription updates the photo description (only if not deleted).
func (p *Photo) UpdateDescription(desc string) error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	p.description = &desc
	p.updatedAt = time.Now()
	return nil
}

// UpdateTags updates the photo tags (only if not deleted).
func (p *Photo) UpdateTags(tags []string) error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	p.tags = tags
	p.updatedAt = time.Now()
	return nil
}

// UpdateLaneCode updates the lane code (only if not deleted).
func (p *Photo) UpdateLaneCode(code string) error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	if !IsValidLaneCode(code) {
		return ErrInvalidLaneCode
	}
	p.laneCode = code
	p.updatedAt = time.Now()
	return nil
}

// SetSurveyYear updates the survey year (only if not deleted).
func (p *Photo) SetSurveyYear(year int) error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	// Validate year is reasonable (2000 to current year + 1)
	currentYear := time.Now().Year()
	if year < 2000 || year > currentYear+1 {
		return errors.New("survey year must be between 2000 and current year + 1")
	}
	p.surveyYear = year
	p.updatedAt = time.Now()
	return nil
}

// SetGCSObjectName updates the GCS object name (only if not deleted).
func (p *Photo) SetGCSObjectName(objectName string) error {
	if p.IsDeleted() {
		return ErrPhotoDeleted
	}
	if objectName == "" {
		return errors.New("GCS object name cannot be empty")
	}
	p.gcsObjectName = objectName
	p.updatedAt = time.Now()
	return nil
}

// SoftDelete marks the photo as deleted.
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

// Restore un-deletes a soft-deleted photo (admin only).
func (p *Photo) Restore() error {
	if !p.IsDeleted() {
		return ErrPhotoNotDeleted
	}
	p.deletedAt = nil
	p.deletedBy = nil
	p.updatedAt = time.Now()
	return nil
}

// PhotoRowParams contains all fields needed to reconstruct a Photo from database row.
// For use by repository layer only.
type PhotoRowParams struct {
	ID               vo.PhotoID
	RouteID          string
	LaneCode         string
	SurveyYear       int
	Latitude         *float64
	Longitude        *float64
	StaValue         *float64
	StaSource        *vo.STASource
	GCSObjectName    string
	FileFormat       vo.FileFormat
	FileSizeBytes    int64
	OriginalFilename *string
	Description      *string
	Tags             []string
	UploadToken      vo.UploadToken
	UploadStatus     vo.UploadStatus
	RetryCount       int
	UploadedBy       string
	UploadedAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	DeletedBy        *string
}

// NewPhotoFromRepository reconstructs a Photo from database row data.
// This bypasses normal validation for DB reconstruction and is intended
// for use by the repository layer only.
// Note: This function assumes the data coming from the database is valid.
func NewPhotoFromRepository(params PhotoRowParams) *Photo {
	var coords *vo.Coordinates
	if params.Latitude != nil && params.Longitude != nil {
		c, _ := vo.NewCoordinates(*params.Latitude, *params.Longitude)
		coords = &c
	}

	return &Photo{
		id:               params.ID,
		routeID:          params.RouteID,
		laneCode:         params.LaneCode,
		surveyYear:       params.SurveyYear,
		coordinates:      coords,
		staValue:         params.StaValue,
		staSource:        params.StaSource,
		gcsObjectName:    params.GCSObjectName,
		fileFormat:       params.FileFormat,
		fileSizeBytes:    params.FileSizeBytes,
		originalFilename: params.OriginalFilename,
		description:      params.Description,
		tags:             params.Tags,
		uploadToken:      params.UploadToken,
		uploadStatus:     params.UploadStatus,
		retryCount:       params.RetryCount,
		uploadedBy:       params.UploadedBy,
		uploadedAt:       params.UploadedAt,
		createdAt:        params.CreatedAt,
		updatedAt:        params.UpdatedAt,
		deletedAt:        params.DeletedAt,
		deletedBy:        params.DeletedBy,
	}
}
