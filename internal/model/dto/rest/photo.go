package rest

import (
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
)

// PhotoResponse - Full photo metadata response.
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

// UpdatePhotoRequest - Update photo metadata.
type UpdatePhotoRequest struct {
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	LaneCode    *string  `json:"lane_code,omitempty"`
}

// Validate validates the update photo request.
func (r *UpdatePhotoRequest) Validate() error {
	if r.LaneCode != nil && !IsValidLaneCode(*r.LaneCode) {
		return model.NewValidationError("lane_code", "lane_code must be in format L1-L10 or R1-R10")
	}
	return nil
}

// UpdatePhotoResponse - Confirmation of update.
type UpdatePhotoResponse struct {
	PhotoID     vo.PhotoID `json:"photo_id"`
	Description *string    `json:"description,omitempty"`
	Tags        []string   `json:"tags"`
	LaneCode    string     `json:"lane_code"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// IsValidLaneCode validates that the lane code matches the format L1-L10 or R1-R10
func IsValidLaneCode(code string) bool {
	return model.LaneCodeRegex.MatchString(code)
}

// DeletePhotoResponse - Confirmation of deletion.
type DeletePhotoResponse struct {
	PhotoID      vo.PhotoID `json:"photo_id"`
	DeletedAt    time.Time  `json:"deleted_at"`
	DeletionType string     `json:"deletion_type"` // "soft" or "hard"
}
