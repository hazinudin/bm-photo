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
