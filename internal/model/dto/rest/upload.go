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

// PhotoAttributes contains photo metadata for upload URL request.
type PhotoAttributes struct {
	RouteID     string   `json:"route_id"`
	LaneCode    string   `json:"lane_code"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	STAValue    *float64 `json:"sta_value,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// GetSignedUploadURLRequest - Phase 1: Request signed upload URL.
type GetSignedUploadURLRequest struct {
	FileMetadata    FileMetadata    `json:"file_metadata"`
	PhotoAttributes PhotoAttributes `json:"photo_attributes"`
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
	if r.PhotoAttributes.RouteID == "" {
		return model.NewValidationError("photo_attributes.route_id", "route_id is required")
	}
	if r.PhotoAttributes.LaneCode != "" && !IsValidLaneCode(r.PhotoAttributes.LaneCode) {
		return model.NewValidationError("photo_attributes.lane_code", "lane_code must be in format L1-L10 or R1-R10")
	}
	if r.PhotoAttributes.Latitude < -90 || r.PhotoAttributes.Latitude > 90 {
		return model.NewValidationError("photo_attributes.latitude", "latitude must be between -90 and 90")
	}
	if r.PhotoAttributes.Longitude < -180 || r.PhotoAttributes.Longitude > 180 {
		return model.NewValidationError("photo_attributes.longitude", "longitude must be between -180 and 180")
	}
	if r.PhotoAttributes.STAValue != nil && *r.PhotoAttributes.STAValue < 0 {
		return model.NewValidationError("photo_attributes.sta_value", "sta_value must be greater than or equal to 0")
	}
	return nil
}

func isValidContentType(ct string) bool {
	return ct == "image/jpeg" || ct == "image/jpg" || ct == "image/png"
}

// GetSignedUploadURLResponse - Phase 1: Response with signed URL.
type GetSignedUploadURLResponse struct {
	UploadToken vo.UploadToken `json:"upload_token"`
	SignedURL   string         `json:"signed_url"`
	PhotoID     vo.PhotoID     `json:"photo_id"`
	ExpiresAt   time.Time      `json:"expires_at"`
}

// ConfirmUploadRequest - Phase 2: Confirm upload completion.
type ConfirmUploadRequest struct {
	UploadToken vo.UploadToken `json:"upload_token"`
}

// Validate validates the confirm upload request.
func (r *ConfirmUploadRequest) Validate() error {
	if !r.UploadToken.IsValid() {
		return model.NewValidationError("upload_token", "invalid upload token format")
	}
	return nil
}

// ConfirmUploadResponse - Phase 2: Confirmation response.
type ConfirmUploadResponse struct {
	PhotoID vo.PhotoID `json:"photo_id"`
	Message string     `json:"message"`
}

// GetNewSignedURLResponse - Response for retry endpoint with new signed URL.
type GetNewSignedURLResponse struct {
	PhotoID     vo.PhotoID     `json:"photo_id"`
	UploadToken vo.UploadToken `json:"upload_token"`
	SignedURL   string         `json:"signed_url"`
	ExpiresAt   time.Time      `json:"expires_at"`
	RetryCount  int            `json:"retry_count"`
	MaxRetries  int            `json:"max_retries"`
}
