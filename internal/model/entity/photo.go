package entity

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/lithammer/shortuuid/v4"
)

// Photo represents a survey photo in the catalog.
// It is the aggregate root and manages its upload lifecycle.
type Photo struct {
	// Identity
	id vo.PhotoID

	// Location attributes
	routeID     string
	laneCode    string
	coordinates vo.Coordinates

	// Linear Reference System
	staValue  float64
	staSource vo.STASource

	// Storage paths
	gcsObjectName       string
	thumbnailSmallPath  *string
	thumbnailMediumPath *string
	thumbnailLargePath  *string

	// File metadata
	fileFormat       vo.FileFormat
	fileSizeBytes    int64
	originalFilename *string

	// EXIF data stored as JSON
	exifData *EXIFData

	// User-provided metadata
	description *string
	tags        []string

	// Upload metadata
	uploadToken  vo.UploadToken
	uploadStatus vo.UploadStatus
	uploadedBy   string // API Key ID (string, references auth)
	uploadedAt   time.Time

	// Processing status
	status vo.PhotoStatus

	// Timestamps
	createdAt             time.Time
	updatedAt             time.Time
	processingCompletedAt *time.Time

	// Soft delete
	deletedAt *time.Time
	deletedBy *string // API Key ID
}

// EXIFData contains extracted EXIF metadata
type EXIFData struct {
	Timestamp    *time.Time `json:"timestamp,omitempty"`
	CameraMake   *string    `json:"camera_make,omitempty"`
	CameraModel  *string    `json:"camera_model,omitempty"`
	GPSLatitude  *float64   `json:"gps_latitude,omitempty"`
	GPSLongitude *float64   `json:"gps_longitude,omitempty"`
	Altitude     *float64   `json:"altitude,omitempty"`
	Orientation  *int       `json:"orientation,omitempty"`
}

// PhotoParams contains parameters for creating a new Photo.
type PhotoParams struct {
	RouteID  string
	LaneCode string

	FileFormat       vo.FileFormat
	FileSizeBytes    int64
	OriginalFilename *string

	UploadToken vo.UploadToken
	UploadedBy  string // API Key ID
}

// ThumbnailPaths contains paths for generated thumbnails
type ThumbnailPaths struct {
	Small  string
	Medium string
	Large  string
}

var (
	ErrInvalidRouteID  = errors.New("route_id is required")
	ErrInvalidLaneCode = errors.New("lane_code must be in format L1-L10 or R1-R10")
	ErrInvalidSTAValue = errors.New("sta_value must be greater than or equal to 0")
	ErrInvalidFileSize = errors.New("file_size_bytes must be greater than 0")
	ErrPhotoDeleted    = errors.New("photo has been deleted")
	ErrInvalidAPIKeyID = errors.New("invalid API key ID")

	ErrUploadNotPending   = errors.New("upload is not in pending state")
	ErrUploadNotCompleted = errors.New("upload must be completed before processing")
	ErrPhotoNotDeleted    = errors.New("photo is not deleted")
)

// NewPhoto creates a new Photo entity with validation.
// The original path is generated automatically using the naming convention: <route_id>_<year>_<sta>_<lane>
func NewPhoto(params PhotoParams) (*Photo, error) {
	// Validate required fields
	if params.RouteID == "" {
		return nil, ErrInvalidRouteID
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

	photo := &Photo{
		id:               vo.NewPhotoID(),
		routeID:          params.RouteID,
		laneCode:         params.LaneCode,
		fileFormat:       params.FileFormat,
		fileSizeBytes:    params.FileSizeBytes,
		originalFilename: params.OriginalFilename,
		uploadToken:      params.UploadToken,
		uploadStatus:     vo.UploadStatusPending,
		uploadedBy:       params.UploadedBy,
		uploadedAt:       now,
		status:           vo.PhotoStatusProcessing,
		createdAt:        now,
		updatedAt:        now,
		tags:             []string{},
	}

	// Generate GCS object name dynamically using naming convention
	photo.gcsObjectName = photo.GenerateGCSObjectName()

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

// Latitude returns the latitude coordinate
func (p *Photo) Latitude() float64 {
	return p.coordinates.Latitude()
}

// Longitude returns the longitude coordinate
func (p *Photo) Longitude() float64 {
	return p.coordinates.Longitude()
}

// STAValue returns the STA value
func (p *Photo) STAValue() float64 {
	return p.staValue
}

// STASource returns the STA source
func (p *Photo) STASource() vo.STASource {
	return p.staSource
}

// GCSObjectName returns the GCS object name
func (p *Photo) GCSObjectName() string {
	return p.gcsObjectName
}

// ThumbnailSmallPath returns the small thumbnail path
func (p *Photo) ThumbnailSmallPath() *string {
	return p.thumbnailSmallPath
}

// ThumbnailMediumPath returns the medium thumbnail path
func (p *Photo) ThumbnailMediumPath() *string {
	return p.thumbnailMediumPath
}

// ThumbnailLargePath returns the large thumbnail path
func (p *Photo) ThumbnailLargePath() *string {
	return p.thumbnailLargePath
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

// EXIFData returns the EXIF data
func (p *Photo) EXIFData() *EXIFData {
	return p.exifData
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

// Status returns the photo status
func (p *Photo) Status() vo.PhotoStatus {
	return p.status
}

// CreatedAt returns the creation timestamp
func (p *Photo) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update timestamp
func (p *Photo) UpdatedAt() time.Time {
	return p.updatedAt
}

// ProcessingCompletedAt returns the processing completion timestamp
func (p *Photo) ProcessingCompletedAt() *time.Time {
	return p.processingCompletedAt
}

// DeletedAt returns the deletion timestamp
func (p *Photo) DeletedAt() *time.Time {
	return p.deletedAt
}

// DeletedBy returns the API key ID that deleted the photo
func (p *Photo) DeletedBy() *string {
	return p.deletedBy
}

// IsReady returns true if photo processing is complete
func (p *Photo) IsReady() bool {
	return p.status == vo.PhotoStatusReady
}

// IsProcessing returns true if photo is still being processed
func (p *Photo) IsProcessing() bool {
	return p.status == vo.PhotoStatusProcessing
}

// IsFailed returns true if photo processing failed
func (p *Photo) IsFailed() bool {
	return p.status == vo.PhotoStatusFailed
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

// Business Methods

// MarkUploadComplete transitions upload status to uploaded.
// Called after GCS upload is verified.
func (p *Photo) MarkUploadComplete() error {
	if p.uploadStatus != vo.UploadStatusPending {
		return ErrUploadNotPending
	}
	p.uploadStatus = vo.UploadStatusUploaded
	p.updatedAt = time.Now()
	return nil
}

// MarkProcessingComplete transitions photo to ready status.
func (p *Photo) MarkProcessingComplete(thumbnailPaths ThumbnailPaths) error {
	if p.uploadStatus != vo.UploadStatusUploaded {
		return ErrUploadNotCompleted
	}
	now := time.Now()
	p.thumbnailSmallPath = &thumbnailPaths.Small
	p.thumbnailMediumPath = &thumbnailPaths.Medium
	p.thumbnailLargePath = &thumbnailPaths.Large
	p.status = vo.PhotoStatusReady
	p.processingCompletedAt = &now
	p.uploadStatus = vo.UploadStatusCompleted
	p.updatedAt = now
	return nil
}

// MarkProcessingFailed transitions photo to failed status.
func (p *Photo) MarkProcessingFailed(reason string) error {
	p.status = vo.PhotoStatusFailed
	p.updatedAt = time.Now()
	return nil
}

// SetEXIFData sets the EXIF metadata.
func (p *Photo) SetEXIFData(exif *EXIFData) {
	p.exifData = exif
	p.updatedAt = time.Now()
}

// SetSTA sets the STA value and source.
func (p *Photo) SetSTA(value float64, source vo.STASource) error {
	if value < 0 {
		return model.ErrInvalidSTAValue
	}
	if !source.IsValid() {
		return vo.ErrInvalidSTASource
	}
	p.staValue = value
	p.staSource = source
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
	p.coordinates = coords
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

// GenerateGCSObjectName generates the GCS object name using the format:
// photos/{year}/{route_id}/{route_id}_{year}_{lane_code}_{shortuuid}.{ext}
// Example: photos/2024/NR-001/NR-001_2024_L1_a1b2c3d4.jpg
func (p *Photo) GenerateGCSObjectName() string {
	year := p.uploadedAt.Year()
	ext := strings.ToLower(p.fileFormat.String())
	baseName := fmt.Sprintf("%s_%d_%s_%s", p.routeID, year, p.laneCode, shortuuid.New())
	return fmt.Sprintf("photos/%d/%s/%s.%s", year, p.routeID, baseName, ext)
}

// GenerateThumbnailPaths generates GCS paths for thumbnails using the gcsObjectName base.
// Format: photos/{year}/{route_id}/{base_name}_small.{ext}
// where base_name is {route_id}_{year}_{lane_code}_{shortuuid} from gcsObjectName
func (p *Photo) GenerateThumbnailPaths() ThumbnailPaths {
	base := strings.TrimSuffix(p.gcsObjectName, path.Ext(p.gcsObjectName))
	ext := strings.ToLower(p.fileFormat.String())

	return ThumbnailPaths{
		Small:  base + "_small." + ext,
		Medium: base + "_medium." + ext,
		Large:  base + "_large." + ext,
	}
}

// PhotoRowParams contains all fields needed to reconstruct a Photo from database row.
// For use by repository layer only.
type PhotoRowParams struct {
	ID                    vo.PhotoID
	RouteID               string
	LaneCode              string
	Latitude              float64
	Longitude             float64
	StaValue              float64
	StaSource             vo.STASource
	GCSObjectName         string
	ThumbnailSmallPath    *string
	ThumbnailMediumPath   *string
	ThumbnailLargePath    *string
	FileFormat            vo.FileFormat
	FileSizeBytes         int64
	OriginalFilename      *string
	EXIFData              *EXIFData
	Description           *string
	Tags                  []string
	UploadToken           vo.UploadToken
	UploadStatus          vo.UploadStatus
	UploadedBy            string
	UploadedAt            time.Time
	Status                vo.PhotoStatus
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ProcessingCompletedAt *time.Time
	DeletedAt             *time.Time
	DeletedBy             *string
}

// NewPhotoFromRepository reconstructs a Photo from database row data.
// This bypasses normal validation for DB reconstruction and is intended
// for use by the repository layer only.
// Note: This function assumes the data coming from the database is valid.
func NewPhotoFromRepository(params PhotoRowParams) *Photo {
	// Create coordinates - using direct struct literal since we control the values
	coords := vo.Coordinates{}
	// Use reflection-free approach: create via setter-like pattern
	// Since Coordinates has private fields, we need to use a workaround
	// The safest approach is to use the constructor which validates
	coords, _ = vo.NewCoordinates(params.Latitude, params.Longitude)

	return &Photo{
		id:                    params.ID,
		routeID:               params.RouteID,
		laneCode:              params.LaneCode,
		coordinates:           coords,
		staValue:              params.StaValue,
		staSource:             params.StaSource,
		gcsObjectName:         params.GCSObjectName,
		thumbnailSmallPath:    params.ThumbnailSmallPath,
		thumbnailMediumPath:   params.ThumbnailMediumPath,
		thumbnailLargePath:    params.ThumbnailLargePath,
		fileFormat:            params.FileFormat,
		fileSizeBytes:         params.FileSizeBytes,
		originalFilename:      params.OriginalFilename,
		exifData:              params.EXIFData,
		description:           params.Description,
		tags:                  params.Tags,
		uploadToken:           params.UploadToken,
		uploadStatus:          params.UploadStatus,
		uploadedBy:            params.UploadedBy,
		uploadedAt:            params.UploadedAt,
		status:                params.Status,
		createdAt:             params.CreatedAt,
		updatedAt:             params.UpdatedAt,
		processingCompletedAt: params.ProcessingCompletedAt,
		deletedAt:             params.DeletedAt,
		deletedBy:             params.DeletedBy,
	}
}
