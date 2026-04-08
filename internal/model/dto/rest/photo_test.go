package rest

import (
	"testing"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePhotoRequest_Validate_ValidRequestWithAllFields(t *testing.T) {
	description := "Updated description"
	tags := []string{"tag1", "tag2"}
	laneCode := "R2"

	req := &UpdatePhotoRequest{
		Description: &description,
		Tags:        tags,
		LaneCode:    &laneCode,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_ValidRequestWithNilFields(t *testing.T) {
	req := &UpdatePhotoRequest{
		Description: nil,
		Tags:        nil,
		LaneCode:    nil,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_ValidRequestWithEmptyTags(t *testing.T) {
	description := "Updated description"

	req := &UpdatePhotoRequest{
		Description: &description,
		Tags:        []string{},
		LaneCode:    nil,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_InvalidLaneFormat(t *testing.T) {
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
			req := &UpdatePhotoRequest{
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

func TestUpdatePhotoRequest_Validate_ValidLaneCode(t *testing.T) {
	tests := []struct {
		name     string
		laneCode string
	}{
		{name: "left lane 1", laneCode: "L1"},
		{name: "left lane 10", laneCode: "L10"},
		{name: "right lane 1", laneCode: "R1"},
		{name: "right lane 10", laneCode: "R10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &UpdatePhotoRequest{
				LaneCode: &tt.laneCode,
			}

			err := req.Validate()

			assert.NoError(t, err)
		})
	}
}

func TestUpdatePhotoRequest_Validate_OnlyDescription(t *testing.T) {
	description := "Only updating description"

	req := &UpdatePhotoRequest{
		Description: &description,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_OnlyTags(t *testing.T) {
	tags := []string{"road", "bridge", "survey"}

	req := &UpdatePhotoRequest{
		Tags: tags,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_OnlyLaneCode(t *testing.T) {
	laneCode := "R3"

	req := &UpdatePhotoRequest{
		LaneCode: &laneCode,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_ValidCoordinates(t *testing.T) {
	lat := -6.2088
	lon := 106.8456

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_ValidCoordinatesAndSTA(t *testing.T) {
	lat := -6.2088
	lon := 106.8456
	sta := 1234.5

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
		STAValue:  &sta,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestUpdatePhotoRequest_Validate_OnlyLatitude(t *testing.T) {
	lat := -6.2088

	req := &UpdatePhotoRequest{
		Latitude: &lat,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "coordinates", validationErr.Field)
}

func TestUpdatePhotoRequest_Validate_OnlyLongitude(t *testing.T) {
	lon := 106.8456

	req := &UpdatePhotoRequest{
		Longitude: &lon,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "coordinates", validationErr.Field)
}

func TestUpdatePhotoRequest_Validate_InvalidLatitudeTooHigh(t *testing.T) {
	lat := 91.0
	lon := 106.8456

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "latitude", validationErr.Field)
	assert.Equal(t, "latitude must be between -90 and 90", validationErr.Message)
}

func TestUpdatePhotoRequest_Validate_InvalidLatitudeTooLow(t *testing.T) {
	lat := -91.0
	lon := 106.8456

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "latitude", validationErr.Field)
	assert.Equal(t, "latitude must be between -90 and 90", validationErr.Message)
}

func TestUpdatePhotoRequest_Validate_InvalidLongitudeTooHigh(t *testing.T) {
	lat := -6.2088
	lon := 181.0

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "longitude", validationErr.Field)
	assert.Equal(t, "longitude must be between -180 and 180", validationErr.Message)
}

func TestUpdatePhotoRequest_Validate_InvalidLongitudeTooLow(t *testing.T) {
	lat := -6.2088
	lon := -181.0

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "longitude", validationErr.Field)
	assert.Equal(t, "longitude must be between -180 and 180", validationErr.Message)
}

func TestUpdatePhotoRequest_Validate_InvalidSTAValueNegative(t *testing.T) {
	lat := -6.2088
	lon := 106.8456
	sta := -1.0

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
		STAValue:  &sta,
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "sta_value", validationErr.Field)
	assert.Equal(t, "sta_value must be greater than or equal to 0", validationErr.Message)
}

func TestUpdatePhotoRequest_Validate_ValidBoundaryLatitude(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
	}{
		{name: "exactly -90", lat: -90.0},
		{name: "exactly 90", lat: 90.0},
		{name: "exactly 0", lat: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lon := 106.8456
			req := &UpdatePhotoRequest{
				Latitude:  &tt.lat,
				Longitude: &lon,
			}

			err := req.Validate()

			assert.NoError(t, err)
		})
	}
}

func TestUpdatePhotoRequest_Validate_ValidBoundaryLongitude(t *testing.T) {
	tests := []struct {
		name string
		lon  float64
	}{
		{name: "exactly -180", lon: -180.0},
		{name: "exactly 180", lon: 180.0},
		{name: "exactly 0", lon: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat := -6.2088
			req := &UpdatePhotoRequest{
				Latitude:  &lat,
				Longitude: &tt.lon,
			}

			err := req.Validate()

			assert.NoError(t, err)
		})
	}
}

func TestUpdatePhotoRequest_Validate_ValidSTAZero(t *testing.T) {
	lat := -6.2088
	lon := 106.8456
	sta := 0.0

	req := &UpdatePhotoRequest{
		Latitude:  &lat,
		Longitude: &lon,
		STAValue:  &sta,
	}

	err := req.Validate()

	assert.NoError(t, err)
}
