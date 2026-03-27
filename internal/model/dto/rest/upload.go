package rest

import (
	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"time"
)

// FileMetadata contains file information for upload URL request.
type FileMetadata struct {
	Filename      string `json:"filename"`
	ContentType   string `json:"content_type"`
	FileSizeBytes int64  `json:"file_size_bytes"`
}

// GetSignedUploadURLRequest - Phase 1: Request signed upload URL.
type GetSignedUploadURLRequest struct {
	FileMetadata FileMetadata `json:"file_metadata"`
}

// Validate validates the upload URL request.
func (r *GetSignedUploadURLRequest) Validate() error {
	if r.FileMetadata.Filename == "" {
		return model.NewValidationError("file_metadata.filename", "filename is required")
	}
	if r.FileMetadata.ContentType == "" {
		return model.NewValidationError("file_metadata.content_type", "content_type is required")
	}
	if !isValidContentType(r.FileMetadata.ContentType) {
		return model.NewValidationError("file_metadata.content_type", "must be image/jpeg or image/png")
	}
	if r.FileMetadata.FileSizeBytes <= 0 {
		return model.NewValidationError("file_metadata.file_size_bytes", "file_size_bytes must be greater than 0")
	}
	if r.FileMetadata.FileSizeBytes > model.MaxFileSizeBytes {
		return model.NewValidationError("file_metadata.file_size_bytes", "file size exceeds maximum of 10MB")
	}
	return nil
}

func isValidContentType(ct string) bool {
	return ct == "image/jpeg" || ct == "image/jpg" || ct == "image/png"
}

// GetSignedUploadURLResponse - Phase 1: Response with signed URL.
type GetSignedUploadURLResponse struct {
	UploadToken   vo.UploadToken `json:"upload_token"`
	SignedURL     string         `json:"signed_url"`
	GCSObjectName string         `json:"gcs_object_name"`
	PhotoID       vo.PhotoID     `json:"photo_id"`
	ExpiresAt     time.Time      `json:"expires_at"`
}

// CompleteUploadRequest - Phase 2: Complete upload with metadata.
type CompleteUploadRequest struct {
	UploadToken     vo.UploadToken `json:"upload_token"`
	RouteID         string         `json:"route_id"`
	LaneCode        string         `json:"lane_code"`
	Latitude        float64        `json:"latitude"`
	Longitude       float64        `json:"longitude"`
	STAValue        *float64       `json:"sta_value,omitempty"`
	Description     *string        `json:"description,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	UploadTimestamp string         `json:"upload_timestamp"`
}

// Validate validates the complete upload request.
func (r *CompleteUploadRequest) Validate() error {
	if !r.UploadToken.IsValid() {
		return model.NewValidationError("upload_token", "invalid upload token format")
	}
	if r.RouteID == "" {
		return model.NewValidationError("route_id", "route_id is required")
	}
	if !IsValidLaneCode(r.LaneCode) {
		return model.NewValidationError("lane_code", "lane_code must be in format L1-L10 or R1-R10")
	}
	if r.Latitude < -90 || r.Latitude > 90 {
		return model.NewValidationError("latitude", "latitude must be between -90 and 90")
	}
	if r.Longitude < -180 || r.Longitude > 180 {
		return model.NewValidationError("longitude", "longitude must be between -180 and 180")
	}
	if r.STAValue != nil && *r.STAValue < 0 {
		return model.NewValidationError("sta_value", "sta_value must be greater than or equal to 0")
	}
	if r.UploadTimestamp == "" {
		return model.NewValidationError("upload_timestamp", "upload_timestamp is required")
	}
	return nil
}

// CompleteUploadResponse - Phase 2: Confirmation response.
type CompleteUploadResponse struct {
	PhotoID       vo.PhotoID    `json:"photo_id"`
	RouteID       string        `json:"route_id"`
	LaneCode      string        `json:"lane_code"`
	Latitude      float64       `json:"latitude"`
	Longitude     float64       `json:"longitude"`
	STAValue      float64       `json:"sta_value"`
	STASource     vo.STASource  `json:"sta_source"`
	FileFormat    vo.FileFormat `json:"file_format"`
	FileSizeBytes int64         `json:"file_size_bytes"`
	Status        string        `json:"status"`
	UploadedAt    time.Time     `json:"uploaded_at"`
	ThumbnailURLs ThumbnailURLs `json:"thumbnail_urls"`
	Message       string        `json:"message"`
}

// ThumbnailURLs contains URLs for photo thumbnails.
type ThumbnailURLs struct {
	Small  *string `json:"small,omitempty"`
	Medium *string `json:"medium,omitempty"`
	Large  *string `json:"large,omitempty"`
}

// PhotoStatusResponse - Get upload/processing status.
type PhotoStatusResponse struct {
	PhotoID         vo.PhotoID `json:"photo_id"`
	Status          string     `json:"status"`
	ThumbnailsReady bool       `json:"thumbnails_ready"`
	EXIFExtracted   bool       `json:"exif_extracted"`
	STACalculated   bool       `json:"sta_calculated"`
	ErrorMessage    *string    `json:"error_message,omitempty"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
}
