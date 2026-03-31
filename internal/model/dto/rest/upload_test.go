package rest

import (
	"testing"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/stretchr/testify/assert"
)

func TestGetSignedUploadURLRequest_Validate_ValidRequest(t *testing.T) {
	staValue := 10.5
	description := "Test photo"

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
				PhotoAttributes: PhotoAttributes{
					RouteID:     "NR-001",
					LaneCode:    "L1",
					Latitude:    -6.2088,
					Longitude:   106.8456,
					STAValue:    &staValue,
					Description: &description,
					Tags:        []string{"test"},
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
		PhotoAttributes: PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
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
		PhotoAttributes: PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
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
				PhotoAttributes: PhotoAttributes{
					RouteID:   "NR-001",
					LaneCode:  "L1",
					Latitude:  -6.2088,
					Longitude: 106.8456,
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
				PhotoAttributes: PhotoAttributes{
					RouteID:   "NR-001",
					LaneCode:  "L1",
					Latitude:  -6.2088,
					Longitude: 106.8456,
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
		PhotoAttributes: PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "file_metadata.file_size_bytes", validationErr.Field)
}

func TestGetSignedUploadURLRequest_Validate_MissingRouteID(t *testing.T) {
	req := &GetSignedUploadURLRequest{
		FileMetadata: FileMetadata{
			Filename:      "test.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024,
		},
		PhotoAttributes: PhotoAttributes{
			RouteID:   "",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "photo_attributes.route_id", validationErr.Field)
}

func TestGetSignedUploadURLRequest_Validate_InvalidLaneFormat(t *testing.T) {
	tests := []struct {
		name     string
		laneCode string
	}{
		{name: "invalid format", laneCode: "X1"},
		{name: "missing number", laneCode: "L"},
		{name: "number out of range low", laneCode: "L0"},
		{name: "number out of range high", laneCode: "L11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GetSignedUploadURLRequest{
				FileMetadata: FileMetadata{
					Filename:      "test.jpg",
					ContentType:   "image/jpeg",
					FileSizeBytes: 1024,
				},
				PhotoAttributes: PhotoAttributes{
					RouteID:   "NR-001",
					LaneCode:  tt.laneCode,
					Latitude:  -6.2088,
					Longitude: 106.8456,
				},
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "photo_attributes.lane_code", validationErr.Field)
		})
	}
}

func TestGetSignedUploadURLRequest_Validate_LatitudeOutOfRange(t *testing.T) {
	tests := []struct {
		name     string
		latitude float64
	}{
		{name: "latitude below -90", latitude: -91.0},
		{name: "latitude above 90", latitude: 91.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GetSignedUploadURLRequest{
				FileMetadata: FileMetadata{
					Filename:      "test.jpg",
					ContentType:   "image/jpeg",
					FileSizeBytes: 1024,
				},
				PhotoAttributes: PhotoAttributes{
					RouteID:   "NR-001",
					LaneCode:  "L1",
					Latitude:  tt.latitude,
					Longitude: 106.8456,
				},
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "photo_attributes.latitude", validationErr.Field)
		})
	}
}

func TestGetSignedUploadURLRequest_Validate_LongitudeOutOfRange(t *testing.T) {
	tests := []struct {
		name      string
		longitude float64
	}{
		{name: "longitude below -180", longitude: -181.0},
		{name: "longitude above 180", longitude: 181.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &GetSignedUploadURLRequest{
				FileMetadata: FileMetadata{
					Filename:      "test.jpg",
					ContentType:   "image/jpeg",
					FileSizeBytes: 1024,
				},
				PhotoAttributes: PhotoAttributes{
					RouteID:   "NR-001",
					LaneCode:  "L1",
					Latitude:  -6.2088,
					Longitude: tt.longitude,
				},
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "photo_attributes.longitude", validationErr.Field)
		})
	}
}

func TestGetSignedUploadURLRequest_Validate_STAValueNegative(t *testing.T) {
	negativeSTA := -1.0

	req := &GetSignedUploadURLRequest{
		FileMetadata: FileMetadata{
			Filename:      "test.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024,
		},
		PhotoAttributes: PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
			STAValue:  &negativeSTA,
		},
	}

	err := req.Validate()

	assert.Error(t, err)
	assert.IsType(t, &model.ValidationError{}, err)
	validationErr := err.(*model.ValidationError)
	assert.Equal(t, "photo_attributes.sta_value", validationErr.Field)
}

func TestConfirmUploadRequest_Validate_ValidRequest(t *testing.T) {
	validToken := vo.NewUploadToken()

	req := &ConfirmUploadRequest{
		UploadToken: validToken,
	}

	err := req.Validate()

	assert.NoError(t, err)
}

func TestConfirmUploadRequest_Validate_InvalidUploadToken(t *testing.T) {
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
			req := &ConfirmUploadRequest{
				UploadToken: tt.token,
			}

			err := req.Validate()

			assert.Error(t, err)
			assert.IsType(t, &model.ValidationError{}, err)
			validationErr := err.(*model.ValidationError)
			assert.Equal(t, "upload_token", validationErr.Field)
		})
	}
}
