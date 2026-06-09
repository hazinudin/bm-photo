package rest

import (
	"fmt"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
)

// PhotoResponse - Full photo metadata response.
type PhotoResponse struct {
	PhotoID          vo.PhotoID    `json:"photo_id"`
	RouteID          string        `json:"route_id"`
	SurveyYear       int           `json:"survey_year"`
	LaneCode         string        `json:"lane_code"`
	Latitude         *float64      `json:"latitude,omitempty"`
	Longitude        *float64      `json:"longitude,omitempty"`
	STAValue         *float64      `json:"sta_value,omitempty"`
	STASource        *vo.STASource `json:"sta_source,omitempty"`
	FileFormat       vo.FileFormat `json:"file_format"`
	FileSizeBytes    int64         `json:"file_size_bytes"`
	OriginalFileName *string       `json:"original_file_name,omitempty"`
	Description      *string       `json:"description,omitempty"`
	Tags             []string      `json:"tags"`
	UploadedAt       time.Time     `json:"uploaded_at"`
	DownloadURL      string        `json:"download_url"`
}

// UpdatePhotoRequest - Update photo metadata.
type UpdatePhotoRequest struct {
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SurveyYear  *int     `json:"survey_year,omitempty"`
	LaneCode    *string  `json:"lane_code,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	STAValue    *float64 `json:"sta_value,omitempty"`
}

// Validate validates the update photo request.
func (r *UpdatePhotoRequest) Validate() error {
	if r.LaneCode != nil && !IsValidLaneCode(*r.LaneCode) {
		return model.NewValidationError("lane_code", "lane_code must be in format L1-L10 or R1-R10")
	}

	hasLat := r.Latitude != nil
	hasLon := r.Longitude != nil
	if hasLat != hasLon {
		return model.NewValidationError("coordinates", "both latitude and longitude must be provided together")
	}

	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90) {
		return model.NewValidationError("latitude", "latitude must be between -90 and 90")
	}

	if r.Longitude != nil && (*r.Longitude < -180 || *r.Longitude > 180) {
		return model.NewValidationError("longitude", "longitude must be between -180 and 180")
	}

	if r.STAValue != nil && *r.STAValue < 0 {
		return model.NewValidationError("sta_value", "sta_value must be greater than or equal to 0")
	}

	if r.SurveyYear != nil {
		minYear := 2000
		maxYear := time.Now().Year() + 1
		if *r.SurveyYear < minYear || *r.SurveyYear > maxYear {
			return model.NewValidationError("survey_year", "survey_year must be between 2000 and current year")
		}
	}

	return nil
}

// UpdatePhotoResponse - Confirmation of update.
type UpdatePhotoResponse struct {
	PhotoID     vo.PhotoID    `json:"photo_id"`
	Description *string       `json:"description,omitempty"`
	Tags        []string      `json:"tags"`
	SurveyYear  int           `json:"survey_year"`
	LaneCode    string        `json:"lane_code"`
	Latitude    *float64      `json:"latitude,omitempty"`
	Longitude   *float64      `json:"longitude,omitempty"`
	STAValue    *float64      `json:"sta_value,omitempty"`
	STASource   *vo.STASource `json:"sta_source,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
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

// BatchUpdateItem - Single item in a batch update request.
type BatchUpdateItem struct {
	PhotoID     string   `json:"photo_id"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SurveyYear  *int     `json:"survey_year,omitempty"`
	LaneCode    *string  `json:"lane_code,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	STAValue    *float64 `json:"sta_value,omitempty"`
}

// BatchUpdateRequest - Request body for batch update endpoint.
type BatchUpdateRequest struct {
	Updates []BatchUpdateItem `json:"updates"`
}

// Validate validates the batch update request.
func (r *BatchUpdateRequest) Validate() error {
	if len(r.Updates) == 0 {
		return model.NewValidationError("updates", "updates array must not be empty")
	}

	if len(r.Updates) > model.MaxBatchUpdateSize {
		return model.NewValidationError("updates", fmt.Sprintf("updates array must not exceed %d items", model.MaxBatchUpdateSize))
	}

	seen := make(map[string]bool, len(r.Updates))
	for _, item := range r.Updates {
		if item.PhotoID == "" {
			return model.NewValidationError("photo_id", "photo_id is required")
		}
		if seen[item.PhotoID] {
			return model.NewValidationError("updates", fmt.Sprintf("duplicate photo_id: %s", item.PhotoID))
		}
		seen[item.PhotoID] = true
	}

	return nil
}

// BatchUpdateItemResult - Result for a single item in a batch update.
type BatchUpdateItemResult struct {
	PhotoID   string               `json:"photo_id"`
	Status    string               `json:"status"` // "success" or "error"
	Error     string               `json:"error,omitempty"`
	ErrorCode string               `json:"error_code,omitempty"`
	Photo     *UpdatePhotoResponse `json:"photo,omitempty"`
}

// BatchUpdateResponse - Response body for batch update endpoint.
type BatchUpdateResponse struct {
	Total     int                   `json:"total"`
	Succeeded int                   `json:"succeeded"`
	Failed    int                   `json:"failed"`
	Results   []BatchUpdateItemResult `json:"results"`
}
