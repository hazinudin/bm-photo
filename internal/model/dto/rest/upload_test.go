package rest

import (
	"testing"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/stretchr/testify/assert"
)

func TestGetSignedUploadURLRequest_Validate_ValidRequest(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "valid JPEG", contentType: "image/jpeg"},
		{name: "valid PNG", contentType: "image/png"},
		{name: "valid JPG", contentType: "image/jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GetSignedUploadURLRequest{
				FileMetadata: FileMetadata{
					Filename:      "test.jpg",
					ContentType:   tt.contentType,
					FileSizeBytes: 1024,
				},
			}

			err := req.Validate()

			assert.NoError(t, err)
		})
	}
}

func TestGetSignedUploadURLRequest_Validate_MissingFilename(t *testing.T) {
	req := &GetSignedUploadURLRequest{
		FileMetadata: FileMetadata{
			Filename:      "",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024,
		},
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "file_metadata.filename", validationErr.Field)
}

func TestGetSignedUploadURLRequest_Validate_MissingContentType(t *testing.T) {
	req := &GetSignedUploadURLRequest{
		FileMetadata: FileMetadata{
			Filename:      "test.jpg",
			ContentType:   "",
			FileSizeBytes: 1024,
		},
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "file_metadata.content_type", validationErr.Field)
}

func TestGetSignedUploadURLRequest_Validate_InvalidContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{name: "invalid gif", contentType: "image/gif"},
		{name: "invalid bmp", contentType: "image/bmp"},
		{name: "invalid text", contentType: "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GetSignedUploadURLRequest{
				FileMetadata: FileMetadata{
					Filename:      "test.jpg",
					ContentType:   tt.contentType,
					FileSizeBytes: 1024,
				},
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "file_metadata.content_type", validationErr.Field)
		})
	}
}

func TestGetSignedUploadURLRequest_Validate_FileSizeLessOrEqualZero(t *testing.T) {
	tests := []struct {
		name          string
		fileSizeBytes int64
	}{
		{name: "zero file size", fileSizeBytes: 0},
		{name: "negative file size", fileSizeBytes: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GetSignedUploadURLRequest{
				FileMetadata: FileMetadata{
					Filename:      "test.jpg",
					ContentType:   "image/jpeg",
					FileSizeBytes: tt.fileSizeBytes,
				},
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "file_metadata.file_size_bytes", validationErr.Field)
		})
	}
}

func TestGetSignedUploadURLRequest_Validate_FileSizeExceedsMax(t *testing.T) {
	req := &GetSignedUploadURLRequest{
		FileMetadata: FileMetadata{
			Filename:      "test.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: model.MaxFileSizeBytes + 1,
		},
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "file_metadata.file_size_bytes", validationErr.Field)
}

func TestCompleteUploadRequest_Validate_ValidRequest(t *testing.T) {
	validToken := vo.NewUploadToken()
	staValue := 10.5
	uploadTimestamp := "2024-01-01T00:00:00Z"

	req := &CompleteUploadRequest{
		UploadToken:     validToken,
		RouteID:         "NR-001",
		LaneCode:        "L1",
		Latitude:        -6.2088,
		Longitude:       106.8456,
		STAValue:        &staValue,
		UploadTimestamp: uploadTimestamp,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestCompleteUploadRequest_Validate_ValidRequestWithNilFields(t *testing.T) {
	validToken := vo.NewUploadToken()

	req := &CompleteUploadRequest{
		UploadToken:     validToken,
		RouteID:         "NR-001",
		LaneCode:        "R5",
		Latitude:        -6.2088,
		Longitude:       106.8456,
		STAValue:        nil,
		UploadTimestamp: "2024-01-01T00:00:00Z",
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestCompleteUploadRequest_Validate_InvalidUploadToken(t *testing.T) {
	tests := []struct {
		name  string
		token vo.UploadToken
	}{
		{name: "empty token", token: vo.UploadToken("")},
		{name: "invalid UUID", token: vo.UploadToken("not-a-uuid")},
		{name: "partial UUID", token: vo.UploadToken("550e8400-e29b")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CompleteUploadRequest{
				UploadToken:     tt.token,
				RouteID:         "NR-001",
				LaneCode:        "L1",
				Latitude:        -6.2088,
				Longitude:       106.8456,
				UploadTimestamp: "2024-01-01T00:00:00Z",
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "upload_token", validationErr.Field)
		})
	}
}

func TestCompleteUploadRequest_Validate_MissingRouteID(t *testing.T) {
	validToken := vo.NewUploadToken()

	req := &CompleteUploadRequest{
		UploadToken:     validToken,
		RouteID:         "",
		LaneCode:        "L1",
		Latitude:        -6.2088,
		Longitude:       106.8456,
		UploadTimestamp: "2024-01-01T00:00:00Z",
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "route_id", validationErr.Field)
}

func TestCompleteUploadRequest_Validate_InvalidLaneFormat(t *testing.T) {
	validToken := vo.NewUploadToken()

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
			req := &CompleteUploadRequest{
				UploadToken:     validToken,
				RouteID:         "NR-001",
				LaneCode:        tt.laneCode,
				Latitude:        -6.2088,
				Longitude:       106.8456,
				UploadTimestamp: "2024-01-01T00:00:00Z",
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "lane_code", validationErr.Field)
		})
	}
}

func TestCompleteUploadRequest_Validate_LatitudeOutOfRange(t *testing.T) {
	validToken := vo.NewUploadToken()

	tests := []struct {
		name     string
		latitude float64
	}{
		{name: "latitude below -90", latitude: -91.0},
		{name: "latitude above 90", latitude: 91.0},
		{name: "latitude at -180", latitude: -180.0},
		{name: "latitude at 180", latitude: 180.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CompleteUploadRequest{
				UploadToken:     validToken,
				RouteID:         "NR-001",
				LaneCode:        "L1",
				Latitude:        tt.latitude,
				Longitude:       106.8456,
				UploadTimestamp: "2024-01-01T00:00:00Z",
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "latitude", validationErr.Field)
		})
	}
}

func TestCompleteUploadRequest_Validate_LongitudeOutOfRange(t *testing.T) {
	validToken := vo.NewUploadToken()

	tests := []struct {
		name      string
		longitude float64
	}{
		{name: "longitude below -180", longitude: -181.0},
		{name: "longitude above 180", longitude: 181.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CompleteUploadRequest{
				UploadToken:     validToken,
				RouteID:         "NR-001",
				LaneCode:        "L1",
				Latitude:        -6.2088,
				Longitude:       tt.longitude,
				UploadTimestamp: "2024-01-01T00:00:00Z",
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "longitude", validationErr.Field)
		})
	}
}

func TestCompleteUploadRequest_Validate_STAValueNegative(t *testing.T) {
	validToken := vo.NewUploadToken()
	negativeSTA := -1.0

	req := &CompleteUploadRequest{
		UploadToken:     validToken,
		RouteID:         "NR-001",
		LaneCode:        "L1",
		Latitude:        -6.2088,
		Longitude:       106.8456,
		STAValue:        &negativeSTA,
		UploadTimestamp: "2024-01-01T00:00:00Z",
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "sta_value", validationErr.Field)
}

func TestCompleteUploadRequest_Validate_MissingUploadTimestamp(t *testing.T) {
	validToken := vo.NewUploadToken()

	req := &CompleteUploadRequest{
		UploadToken:     validToken,
		RouteID:         "NR-001",
		LaneCode:        "L1",
		Latitude:        -6.2088,
		Longitude:       106.8456,
		UploadTimestamp: "",
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "upload_timestamp", validationErr.Field)
}
