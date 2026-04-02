package rest

import (
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
)

// BrowsePhotosRequest - Query photos with filters.
type BrowsePhotosRequest struct {
	RouteID  string   `query:"route_id"`
	STAStart *float64 `query:"sta_start"`
	STAEnd   *float64 `query:"sta_end"`
	LaneCode *string  `query:"lane_code"`
	Page     int      `query:"page"`
	PerPage  int      `query:"per_page"`
}

// Validate validates and sets defaults for browse request.
func (r *BrowsePhotosRequest) Validate() error {
	if r.RouteID == "" {
		return model.NewValidationError("route_id", "route_id is required")
	}
	if r.STAStart != nil && *r.STAStart < 0 {
		return model.NewValidationError("sta_start", "sta_start must be >= 0")
	}
	if r.STAEnd != nil && *r.STAEnd < 0 {
		return model.NewValidationError("sta_end", "sta_end must be >= 0")
	}
	if r.STAStart != nil && r.STAEnd != nil && *r.STAStart > *r.STAEnd {
		return model.NewValidationError("sta_start", "sta_start must be <= sta_end")
	}
	if r.LaneCode != nil && !IsValidLaneCode(*r.LaneCode) {
		return model.NewValidationError("lane_code", "lane_code must be in format L1-L10 or R1-R10")
	}
	if r.Page <= 0 {
		r.Page = model.DefaultPage
	}
	if r.PerPage <= 0 || r.PerPage > model.MaxPerPage {
		r.PerPage = model.DefaultPerPage
	}
	return nil
}

// BrowsePhotosResponse - Paginated photo list.
type BrowsePhotosResponse struct {
	Photos     []PhotoSummary `json:"photos"`
	Pagination Pagination     `json:"pagination"`
}

// PhotoSummary - Summary of a photo for list views.
type PhotoSummary struct {
	PhotoID          vo.PhotoID `json:"photo_id"`
	RouteID          string     `json:"route_id"`
	LaneCode         string     `json:"lane_code"`
	STAValue         *float64   `json:"sta_value,omitempty"`
	GCSURL           string     `json:"gcs_url"`
	UploadedAt       time.Time  `json:"uploaded_at"`
	OriginalFileName string     `json:"file_name"`
}

// Pagination - Pagination information.
type Pagination struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalCount  int64 `json:"total_count"`
	TotalPages  int   `json:"total_pages"`
}

// SearchPhotosRequest - Advanced search with multiple filters.
type SearchPhotosRequest struct {
	RouteIDs   []string   `json:"route_ids"`
	STARanges  []STARange `json:"sta_ranges"`
	LaneCodes  []string   `json:"lane_codes"`
	DateStart  *time.Time `json:"date_start,omitempty"`
	DateEnd    *time.Time `json:"date_end,omitempty"`
	Tags       []string   `json:"tags"`
	HasEXIFGPS *bool      `json:"has_exif_gps,omitempty"`
	Page       int        `json:"page"`
	PerPage    int        `json:"per_page"`
}

// STARange - Range of STA values.
type STARange struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// Validate validates and sets defaults for search request.
func (r *SearchPhotosRequest) Validate() error {
	if r.Page <= 0 {
		r.Page = model.DefaultPage
	}
	if r.PerPage <= 0 || r.PerPage > model.MaxPerPage {
		r.PerPage = model.DefaultPerPage
	}
	return nil
}

// SearchPhotosResponse - Paginated search results.
type SearchPhotosResponse struct {
	Photos     []PhotoSummary `json:"photos"`
	Pagination Pagination     `json:"pagination"`
}
