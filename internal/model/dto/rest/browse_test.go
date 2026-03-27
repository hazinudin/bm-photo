package rest

import (
	"testing"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestBrowsePhotosRequest_Validate_ValidWithAllFilters(t *testing.T) {
	staStart := 0.0
	staEnd := 100.5
	laneCode := "L1"

	req := &BrowsePhotosRequest{
		RouteID:  "NR-001",
		STAStart: &staStart,
		STAEnd:   &staEnd,
		LaneCode: &laneCode,
		Page:     1,
		PerPage:  20,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestBrowsePhotosRequest_Validate_ValidWithMinimalFields(t *testing.T) {
	req := &BrowsePhotosRequest{
		RouteID: "NR-001",
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestBrowsePhotosRequest_Validate_MissingRouteID(t *testing.T) {
	req := &BrowsePhotosRequest{
		RouteID: "",
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "route_id", validationErr.Field)
	assert.Equal(t, "route_id is required", validationErr.Message)
}

func TestBrowsePhotosRequest_Validate_STAStartNegative(t *testing.T) {
	negativeSTA := -1.0

	req := &BrowsePhotosRequest{
		RouteID:  "NR-001",
		STAStart: &negativeSTA,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "sta_start", validationErr.Field)
	assert.Equal(t, "sta_start must be >= 0", validationErr.Message)
}

func TestBrowsePhotosRequest_Validate_STAEndNegative(t *testing.T) {
	negativeSTA := -5.0

	req := &BrowsePhotosRequest{
		RouteID: "NR-001",
		STAEnd:  &negativeSTA,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "sta_end", validationErr.Field)
	assert.Equal(t, "sta_end must be >= 0", validationErr.Message)
}

func TestBrowsePhotosRequest_Validate_STAStartGreaterThanSTAEnd(t *testing.T) {
	staStart := 100.0
	staEnd := 50.0

	req := &BrowsePhotosRequest{
		RouteID:  "NR-001",
		STAStart: &staStart,
		STAEnd:   &staEnd,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "sta_start", validationErr.Field)
	assert.Equal(t, "sta_start must be <= sta_end", validationErr.Message)
}

func TestBrowsePhotosRequest_Validate_STAStartEqualsSTAEnd(t *testing.T) {
	staValue := 50.0

	req := &BrowsePhotosRequest{
		RouteID:  "NR-001",
		STAStart: &staValue,
		STAEnd:   &staValue,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestBrowsePhotosRequest_Validate_InvalidLaneFormat(t *testing.T) {
	tests := []struct {
		name     string
		laneCode string
	}{
		{name: "empty lane", laneCode: ""},
		{name: "invalid format", laneCode: "X1"},
		{name: "missing number", laneCode: "L"},
		{name: "number out of range low", laneCode: "L0"},
		{name: "number out of range high", laneCode: "L11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &BrowsePhotosRequest{
				RouteID:  "NR-001",
				LaneCode: &tt.laneCode,
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "lane_code", validationErr.Field)
			assert.Equal(t, "lane_code must be in format L1-L10 or R1-R10", validationErr.Message)
		})
	}
}

func TestBrowsePhotosRequest_Validate_PageDefaultApplied(t *testing.T) {
	tests := []struct {
		name         string
		inputPage    int
		expectedPage int
	}{
		{name: "zero page defaults to 1", inputPage: 0, expectedPage: model.DefaultPage},
		{name: "negative page defaults to 1", inputPage: -1, expectedPage: model.DefaultPage},
		{name: "positive page remains", inputPage: 5, expectedPage: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &BrowsePhotosRequest{
				RouteID: "NR-001",
				Page:    tt.inputPage,
			}

			err := req.Validate()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPage, req.Page)
		})
	}
}

func TestBrowsePhotosRequest_Validate_PerPageDefaultApplied(t *testing.T) {
	tests := []struct {
		name            string
		inputPerPage    int
		expectedPerPage int
	}{
		{name: "zero per_page defaults to 20", inputPerPage: 0, expectedPerPage: model.DefaultPerPage},
		{name: "negative per_page defaults to 20", inputPerPage: -1, expectedPerPage: model.DefaultPerPage},
		{name: "positive per_page within limit remains", inputPerPage: 50, expectedPerPage: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &BrowsePhotosRequest{
				RouteID: "NR-001",
				PerPage: tt.inputPerPage,
			}

			err := req.Validate()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPerPage, req.PerPage)
		})
	}
}

func TestBrowsePhotosRequest_Validate_PerPageExceedsMax(t *testing.T) {
	req := &BrowsePhotosRequest{
		RouteID: "NR-001",
		PerPage: model.MaxPerPage + 100,
	}

	err := req.Validate()

	assert.NoError(t, err)
	assert.Equal(t, model.DefaultPerPage, req.PerPage)
}

func TestBrowsePhotosRequest_Validate_PerPageAtMax(t *testing.T) {
	req := &BrowsePhotosRequest{
		RouteID: "NR-001",
		PerPage: model.MaxPerPage,
	}

	err := req.Validate()

	assert.NoError(t, err)
	assert.Equal(t, model.MaxPerPage, req.PerPage)
}

func TestSearchPhotosRequest_Validate_ValidRequest(t *testing.T) {
	req := &SearchPhotosRequest{
		RouteIDs:  []string{"NR-001", "NR-002"},
		LaneCodes: []string{"L1", "R2"},
		Page:      1,
		PerPage:   20,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestSearchPhotosRequest_Validate_ValidRequestEmptyFilters(t *testing.T) {
	req := &SearchPhotosRequest{
		RouteIDs:  []string{},
		LaneCodes: nil,
		Page:      1,
		PerPage:   20,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestSearchPhotosRequest_Validate_PageDefaultApplied(t *testing.T) {
	tests := []struct {
		name         string
		inputPage    int
		expectedPage int
	}{
		{name: "zero page defaults to 1", inputPage: 0, expectedPage: model.DefaultPage},
		{name: "negative page defaults to 1", inputPage: -5, expectedPage: model.DefaultPage},
		{name: "positive page remains", inputPage: 10, expectedPage: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &SearchPhotosRequest{
				Page: tt.inputPage,
			}

			err := req.Validate()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPage, req.Page)
		})
	}
}

func TestSearchPhotosRequest_Validate_PerPageDefaultApplied(t *testing.T) {
	tests := []struct {
		name            string
		inputPerPage    int
		expectedPerPage int
	}{
		{name: "zero per_page defaults to 20", inputPerPage: 0, expectedPerPage: model.DefaultPerPage},
		{name: "negative per_page defaults to 20", inputPerPage: -10, expectedPerPage: model.DefaultPerPage},
		{name: "positive per_page within limit remains", inputPerPage: 50, expectedPerPage: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &SearchPhotosRequest{
				PerPage: tt.inputPerPage,
			}

			err := req.Validate()

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPerPage, req.PerPage)
		})
	}
}

func TestSearchPhotosRequest_Validate_PerPageExceedsMax(t *testing.T) {
	req := &SearchPhotosRequest{
		PerPage: model.MaxPerPage + 50,
	}

	err := req.Validate()

	assert.NoError(t, err)
	assert.Equal(t, model.DefaultPerPage, req.PerPage)
}

func TestSearchPhotosRequest_Validate_PerPageAtMax(t *testing.T) {
	req := &SearchPhotosRequest{
		PerPage: model.MaxPerPage,
	}

	err := req.Validate()

	assert.NoError(t, err)
	assert.Equal(t, model.MaxPerPage, req.PerPage)
}
