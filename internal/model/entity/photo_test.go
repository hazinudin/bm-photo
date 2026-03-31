package entity

import (
	"testing"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/stretchr/testify/assert"
)

func validPhotoParams() PhotoParams {
	return PhotoParams{
		RouteID:          "NR-001",
		LaneCode:         "L1",
		FileFormat:       vo.FileFormatJPEG,
		FileSizeBytes:    1024000,
		OriginalFilename: strPtr("photo.jpg"),
		UploadToken:      vo.NewUploadToken(),
		UploadedBy:       "api-key-123",
	}
}

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestNewPhoto_ValidParams_ReturnsPhoto(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.NotNil(t, photo)
	assert.NotEmpty(t, photo.ID().String())
	assert.Equal(t, params.RouteID, photo.RouteID())
	assert.Equal(t, params.FileFormat, photo.FileFormat())
	assert.Equal(t, params.FileSizeBytes, photo.FileSizeBytes())
	assert.Equal(t, params.OriginalFilename, photo.OriginalFilename())
	assert.Equal(t, params.UploadToken, photo.UploadToken())
	assert.Equal(t, params.UploadedBy, photo.UploadedBy())
	assert.Equal(t, vo.UploadStatusPending, photo.UploadStatus())
	assert.Equal(t, vo.PhotoStatusProcessing, photo.Status())
	assert.False(t, photo.IsDeleted())
	assert.True(t, photo.IsProcessing())
	assert.True(t, photo.IsUploadPending())
}

func TestNewPhoto_MissingRouteID_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	params.RouteID = ""
	photo, err := NewPhoto(params)
	assert.Nil(t, photo)
	assert.ErrorIs(t, err, ErrInvalidRouteID)
}

func TestSetLaneCode_InvalidValue_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	tests := []struct {
		name string
		lane string
	}{
		{"empty lane", ""},
		{"invalid format", "X1"},
		{"missing number", "L"},
		{"number out of range low", "L0"},
		{"number out of range high", "L11"},
		{"number out of range high", "R11"},
		{"negative format", "L-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := photo.SetLaneCode(tt.lane)
			assert.ErrorIs(t, err, ErrInvalidLaneCode)
		})
	}
}

func TestSetSTA_InvalidValue_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	tests := []struct {
		name string
		sta  float64
	}{
		{"negative STA", -1.0},
		{"negative STA large", -100.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := photo.SetSTA(tt.sta, vo.STASourceUserProvided)
			assert.ErrorIs(t, err, model.ErrInvalidSTAValue)
		})
	}
}

func TestNewPhoto_InvalidFileSizeBytes_ReturnsError(t *testing.T) {
	tests := []struct {
		name string
		size int64
	}{
		{"zero size", 0},
		{"negative size", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validPhotoParams()
			params.FileSizeBytes = tt.size
			photo, err := NewPhoto(params)
			assert.Nil(t, photo)
			assert.ErrorIs(t, err, ErrInvalidFileSize)
		})
	}
}

func TestNewPhoto_MissingUploadedBy_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	params.UploadedBy = ""
	photo, err := NewPhoto(params)
	assert.Nil(t, photo)
	assert.ErrorIs(t, err, ErrInvalidAPIKeyID)
}

func TestNewPhoto_InvalidFileFormat_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	params.FileFormat = vo.FileFormat("GIF")
	photo, err := NewPhoto(params)
	assert.Nil(t, photo)
	assert.ErrorIs(t, err, vo.ErrInvalidFileFormat)
}

func TestSetCoordinates_InvalidValue_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	tests := []struct {
		name string
		lat  float64
		lon  float64
		err  error
	}{
		{"latitude too low", -91, 0, vo.ErrInvalidLatitude},
		{"latitude too high", 91, 0, vo.ErrInvalidLatitude},
		{"longitude too low", 0, -181, vo.ErrInvalidLongitude},
		{"longitude too high", 0, 181, vo.ErrInvalidLongitude},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := photo.SetCoordinates(tt.lat, tt.lon)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestNewPhoto_InvalidUploadToken_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	params.UploadToken = vo.UploadToken("invalid-token")
	photo, err := NewPhoto(params)
	assert.Nil(t, photo)
	assert.ErrorIs(t, err, vo.ErrInvalidUploadToken)
}

func TestPhoto_Getters(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	assert.NotEmpty(t, photo.ID().String())
	assert.True(t, photo.ID().IsValid())
	assert.Equal(t, params.RouteID, photo.RouteID())
	assert.Nil(t, photo.ThumbnailSmallPath())
	assert.Nil(t, photo.ThumbnailMediumPath())
	assert.Nil(t, photo.ThumbnailLargePath())
	assert.Equal(t, params.FileFormat, photo.FileFormat())
	assert.Equal(t, params.FileSizeBytes, photo.FileSizeBytes())
	assert.Equal(t, params.OriginalFilename, photo.OriginalFilename())
	assert.Nil(t, photo.EXIFData())
	assert.Nil(t, photo.Description())
	assert.Empty(t, photo.Tags())
	assert.Equal(t, params.UploadToken, photo.UploadToken())
	assert.Equal(t, vo.UploadStatusPending, photo.UploadStatus())
	assert.Equal(t, params.UploadedBy, photo.UploadedBy())
	assert.False(t, photo.UploadedAt().IsZero())
	assert.Equal(t, vo.PhotoStatusProcessing, photo.Status())
	assert.False(t, photo.CreatedAt().IsZero())
	assert.False(t, photo.UpdatedAt().IsZero())
	assert.Nil(t, photo.ProcessingCompletedAt())
	assert.Nil(t, photo.DeletedAt())
	assert.Nil(t, photo.DeletedBy())
}

func TestPhoto_IsReady(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.False(t, photo.IsReady())

	paths := ThumbnailPaths{Small: "s", Medium: "m", Large: "l"}
	_ = photo.MarkUploadComplete()
	_ = photo.MarkProcessingComplete(paths)
	assert.True(t, photo.IsReady())
}

func TestPhoto_IsProcessing(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.True(t, photo.IsProcessing())

	paths := ThumbnailPaths{Small: "s", Medium: "m", Large: "l"}
	_ = photo.MarkUploadComplete()
	_ = photo.MarkProcessingComplete(paths)
	assert.False(t, photo.IsProcessing())
}

func TestPhoto_IsFailed(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.False(t, photo.IsFailed())

	_ = photo.MarkProcessingFailed("processing error")
	assert.True(t, photo.IsFailed())
}

func TestPhoto_IsDeleted(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.False(t, photo.IsDeleted())
	assert.Nil(t, photo.DeletedAt())

	_ = photo.SoftDelete("user-123")
	assert.True(t, photo.IsDeleted())
	assert.NotNil(t, photo.DeletedAt())
}

func TestPhoto_IsUploadPending(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.True(t, photo.IsUploadPending())
	assert.False(t, photo.IsUploadCompleted())

	_ = photo.MarkUploadComplete()
	assert.False(t, photo.IsUploadPending())
}

func TestPhoto_IsUploadCompleted(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.False(t, photo.IsUploadCompleted())

	_ = photo.MarkUploadComplete()
	paths := ThumbnailPaths{Small: "s", Medium: "m", Large: "l"}
	_ = photo.MarkProcessingComplete(paths)
	assert.True(t, photo.IsUploadCompleted())
}

func TestPhoto_MarkUploadComplete_PendingStatus_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.Equal(t, vo.UploadStatusPending, photo.UploadStatus())

	err = photo.MarkUploadComplete()
	assert.NoError(t, err)
	assert.Equal(t, vo.UploadStatusUploaded, photo.UploadStatus())
	assert.False(t, photo.IsUploadPending())
}

func TestPhoto_MarkUploadComplete_AlreadyUploaded_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	_ = photo.MarkUploadComplete()

	err = photo.MarkUploadComplete()
	assert.ErrorIs(t, err, ErrUploadNotPending)
}

func TestPhoto_MarkProcessingComplete_UploadedStatus_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	_ = photo.MarkUploadComplete()

	paths := ThumbnailPaths{
		Small:  "thumbnails/small/photo.jpg",
		Medium: "thumbnails/medium/photo.jpg",
		Large:  "thumbnails/large/photo.jpg",
	}

	err = photo.MarkProcessingComplete(paths)
	assert.NoError(t, err)
	assert.Equal(t, vo.PhotoStatusReady, photo.Status())
	assert.Equal(t, vo.UploadStatusCompleted, photo.UploadStatus())
	assert.NotNil(t, photo.ThumbnailSmallPath())
	assert.Equal(t, paths.Small, *photo.ThumbnailSmallPath())
	assert.NotNil(t, photo.ThumbnailMediumPath())
	assert.Equal(t, paths.Medium, *photo.ThumbnailMediumPath())
	assert.NotNil(t, photo.ThumbnailLargePath())
	assert.Equal(t, paths.Large, *photo.ThumbnailLargePath())
	assert.NotNil(t, photo.ProcessingCompletedAt())
	assert.True(t, photo.IsReady())
	assert.True(t, photo.IsUploadCompleted())
}

func TestPhoto_MarkProcessingComplete_PendingStatus_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	paths := ThumbnailPaths{Small: "s", Medium: "m", Large: "l"}
	err = photo.MarkProcessingComplete(paths)
	assert.ErrorIs(t, err, ErrUploadNotCompleted)
}

func TestPhoto_MarkProcessingFailed_SetsStatus(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.Equal(t, vo.PhotoStatusProcessing, photo.Status())

	err = photo.MarkProcessingFailed("image processing failed")
	assert.NoError(t, err)
	assert.Equal(t, vo.PhotoStatusFailed, photo.Status())
	assert.True(t, photo.IsFailed())
}

func TestPhoto_SetEXIFData_UpdatesCorrectly(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.Nil(t, photo.EXIFData())

	now := time.Now()
	cameraMake := "Canon"
	cameraModel := "EOS R5"
	orientation := 1
	alt := 100.0
	lat := -6.2
	lon := 106.8

	exif := &EXIFData{
		Timestamp:    &now,
		CameraMake:   &cameraMake,
		CameraModel:  &cameraModel,
		GPSLatitude:  &lat,
		GPSLongitude: &lon,
		Altitude:     &alt,
		Orientation:  &orientation,
	}

	photo.SetEXIFData(exif)
	assert.NotNil(t, photo.EXIFData())
	assert.Equal(t, &now, photo.EXIFData().Timestamp)
	assert.Equal(t, &cameraMake, photo.EXIFData().CameraMake)
	assert.Equal(t, &cameraModel, photo.EXIFData().CameraModel)
}

func TestPhoto_SetSTA_ValidValues_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	err = photo.SetSTA(25.5, vo.STASourceLRSInterpolated)
	assert.NoError(t, err)
	assert.Equal(t, 25.5, photo.STAValue())
	assert.Equal(t, vo.STASourceLRSInterpolated, photo.STASource())
}

func TestPhoto_SetSTA_NegativeValue_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	err = photo.SetSTA(-1.0, vo.STASourceUserProvided)
	assert.ErrorIs(t, err, model.ErrInvalidSTAValue)
}

func TestPhoto_SetSTA_InvalidSource_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	err = photo.SetSTA(10.0, vo.STASource("invalid"))
	assert.ErrorIs(t, err, vo.ErrInvalidSTASource)
}

func TestPhoto_UpdateDescription_NonDeletedPhoto_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.Nil(t, photo.Description())

	err = photo.UpdateDescription("Survey photo description")
	assert.NoError(t, err)
	assert.NotNil(t, photo.Description())
	assert.Equal(t, "Survey photo description", *photo.Description())
}

func TestPhoto_UpdateDescription_DeletedPhoto_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	_ = photo.SoftDelete("admin")

	err = photo.UpdateDescription("new description")
	assert.ErrorIs(t, err, ErrPhotoDeleted)
}

func TestPhoto_UpdateTags_NonDeletedPhoto_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.Empty(t, photo.Tags())

	tags := []string{"survey", "pavement", "crack"}
	err = photo.UpdateTags(tags)
	assert.NoError(t, err)
	assert.Equal(t, tags, photo.Tags())
}

func TestPhoto_UpdateTags_DeletedPhoto_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	_ = photo.SoftDelete("admin")

	err = photo.UpdateTags([]string{"tag1"})
	assert.ErrorIs(t, err, ErrPhotoDeleted)
}

func TestPhoto_UpdateLaneCode_ValidValue_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.Equal(t, "L1", photo.LaneCode())

	err = photo.UpdateLaneCode("R2")
	assert.NoError(t, err)
	assert.Equal(t, "R2", photo.LaneCode())
}

func TestPhoto_UpdateLaneCode_InvalidValue_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	tests := []struct {
		name string
		lane string
	}{
		{"empty lane", ""},
		{"invalid format", "X1"},
		{"missing number", "L"},
		{"number out of range low", "L0"},
		{"number out of range high", "L11"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := photo.UpdateLaneCode(tt.lane)
			assert.ErrorIs(t, err, ErrInvalidLaneCode)
		})
	}
}

func TestPhoto_UpdateLaneCode_DeletedPhoto_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	_ = photo.SoftDelete("admin")

	err = photo.UpdateLaneCode("R2")
	assert.ErrorIs(t, err, ErrPhotoDeleted)
}

func TestPhoto_SoftDelete_NonDeleted_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.False(t, photo.IsDeleted())
	assert.Nil(t, photo.DeletedAt())
	assert.Nil(t, photo.DeletedBy())

	err = photo.SoftDelete("admin-user")
	assert.NoError(t, err)
	assert.True(t, photo.IsDeleted())
	assert.NotNil(t, photo.DeletedAt())
	assert.NotNil(t, photo.DeletedBy())
	assert.Equal(t, "admin-user", *photo.DeletedBy())
}

func TestPhoto_SoftDelete_AlreadyDeleted_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	_ = photo.SoftDelete("admin")

	err = photo.SoftDelete("another-admin")
	assert.ErrorIs(t, err, ErrPhotoDeleted)
}

func TestPhoto_Restore_DeletedPhoto_Succeeds(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	_ = photo.SoftDelete("admin")
	assert.True(t, photo.IsDeleted())

	err = photo.Restore()
	assert.NoError(t, err)
	assert.False(t, photo.IsDeleted())
	assert.Nil(t, photo.DeletedAt())
	assert.Nil(t, photo.DeletedBy())
}

func TestPhoto_Restore_NonDeletedPhoto_ReturnsError(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)
	assert.False(t, photo.IsDeleted())

	err = photo.Restore()
	assert.ErrorIs(t, err, ErrPhotoNotDeleted)
}

func TestPhoto_GenerateThumbnailPaths_ReturnsCorrectPaths(t *testing.T) {
	params := validPhotoParams()
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	paths := photo.GenerateThumbnailPaths()

	assert.Contains(t, paths.Small, "photos/2026/NR-001/")
	assert.Contains(t, paths.Small, "_small.jpeg")
	assert.Contains(t, paths.Medium, "photos/2026/NR-001/")
	assert.Contains(t, paths.Medium, "_medium.jpeg")
	assert.Contains(t, paths.Large, "photos/2026/NR-001/")
	assert.Contains(t, paths.Large, "_large.jpeg")
}

func TestPhoto_GenerateThumbnailPaths_PNGFormat(t *testing.T) {
	params := validPhotoParams()
	params.FileFormat = vo.FileFormatPNG
	photo, err := NewPhoto(params)
	assert.NoError(t, err)

	paths := photo.GenerateThumbnailPaths()

	assert.Contains(t, paths.Small, ".png")
	assert.Contains(t, paths.Medium, ".png")
	assert.Contains(t, paths.Large, ".png")
}
