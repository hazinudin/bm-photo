package rest

import (
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
)

type PhotoStatsRequest struct {
	RouteID      string `query:"route_id"`
	SurveyYear   *int   `query:"survey_year"`
	UploadedOnly *bool  `query:"uploaded_only"`
}

func (r *PhotoStatsRequest) Validate() error {
	if r.RouteID == "" {
		return model.NewValidationError("route_id", "route_id is required")
	}
	if r.SurveyYear != nil && (*r.SurveyYear < 2000 || *r.SurveyYear > time.Now().Year()+1) {
		return model.NewValidationError("survey_year", "survey_year must be between 2000 and current year + 1")
	}
	return nil
}

type LaneStats struct {
	LaneCode string `json:"lane_code"`
	Count    int64  `json:"count"`
}

type PhotoStatsResponse struct {
	RouteID    string      `json:"route_id"`
	TotalCount int64       `json:"total_count"`
	LaneStats  []LaneStats `json:"lane_stats"`
}
