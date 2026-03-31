package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// UploadServiceImpl implements UploadService for two-phase upload workflow
type UploadServiceImpl struct {
	photoRepo         repository.PhotoRepository
	pendingUploadRepo repository.PendingUploadRepository
	gcsClient         GCSClient
	logger            Logger
}

// NewUploadService creates a new UploadService instance
func NewUploadService(
	photoRepo repository.PhotoRepository,
	pendingUploadRepo repository.PendingUploadRepository,
	gcsClient GCSClient,
	logger Logger,
) *UploadServiceImpl {
	return &UploadServiceImpl{
		photoRepo:         photoRepo,
		pendingUploadRepo: pendingUploadRepo,
		gcsClient:         gcsClient,
		logger:            logger,
	}
}

// generateGCSObjectName generates a GCS object name following the naming convention:
// photos/{year}/{route_id}/{route_id}_{year}_{lane}_{shortuuid}.{ext}
func generateGCSObjectName(routeID, laneCode, contentType string) string {
	year := time.Now().Year()
	shortUUID := vo.NewPhotoID().String()[:8]
	// Derive extension from content type
	ext := "jpg"
	if contentType == "image/png" {
		ext = "png"
	}
	return fmt.Sprintf("photos/%d/%s/%s_%d_%s_%s.%s", year, routeID, routeID, year, laneCode, shortUUID, ext)
}

// GetSignedURL handles Phase 1 of the two-phase upload workflow.
// It validates the request, creates a photo record, generates a signed URL,
// and returns the upload token, signed URL, photo ID, and expiry time.
func (s *UploadServiceImpl) GetSignedURL(
	ctx context.Context,
	req *rest.GetSignedUploadURLRequest,
	apiKeyID string,
) (*rest.GetSignedUploadURLResponse, error) {
	// Validate request DTO
	if err := req.Validate(); err != nil {
		s.logger.Warn("Invalid upload request", map[string]interface{}{
			"api_key_id": apiKeyID,
			"error":      err.Error(),
		})
		return nil, err
	}

	// Check concurrent upload limit
	activeCount, err := s.pendingUploadRepo.CountActiveByAPIKey(ctx, apiKeyID)
	if err != nil {
		s.logger.Error("Failed to count active uploads", err, map[string]interface{}{
			"api_key_id": apiKeyID,
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to check upload quota", err)
	}

	if activeCount >= model.MaxPendingUploadsPerKey {
		s.logger.Warn("Upload quota exceeded", map[string]interface{}{
			"api_key_id":   apiKeyID,
			"active_count": activeCount,
			"max_allowed":  model.MaxPendingUploadsPerKey,
		})
		return nil, ErrUploadQuotaExceeded
	}

	// Parse file format from content type
	fileFormat, err := vo.ParseFileFormatFromContentType(req.FileMetadata.ContentType)
	if err != nil {
		return nil, ErrUnsupportedFormat
	}

	// Generate upload token
	uploadToken := vo.NewUploadToken()

	// Generate GCS object name using the naming convention
	// Route ID and lane may be empty initially, will be updated in confirm phase
	routeID := req.PhotoAttributes.RouteID
	laneCode := req.PhotoAttributes.LaneCode
	if laneCode == "" {
		laneCode = "unknown"
	}
	gcsObjectName := generateGCSObjectName(routeID, laneCode, req.FileMetadata.ContentType)

	// Create photo entity with pending status
	photoParams := entity.PhotoParams{
		RouteID:          routeID,
		LaneCode:         laneCode,
		GCSObjectName:    gcsObjectName,
		FileFormat:       fileFormat,
		FileSizeBytes:    req.FileMetadata.FileSizeBytes,
		OriginalFilename: &req.FileMetadata.Filename,
		UploadToken:      uploadToken,
		UploadedBy:       apiKeyID,
	}

	photo, err := entity.NewPhoto(photoParams)
	if err != nil {
		s.logger.Error("Failed to create photo entity", err, nil)
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to create photo record", err)
	}

	// Set coordinates if provided
	if req.PhotoAttributes.Latitude != 0 || req.PhotoAttributes.Longitude != 0 {
		if err := photo.SetCoordinates(req.PhotoAttributes.Latitude, req.PhotoAttributes.Longitude); err != nil {
			s.logger.Warn("Failed to set coordinates", map[string]interface{}{
				"photo_id": photo.ID().String(),
				"error":    err.Error(),
			})
		}
	}

	// Set STA value if provided
	if req.PhotoAttributes.STAValue != nil {
		// Default STA source to user_provided for user-provided values
		if err := photo.SetSTA(*req.PhotoAttributes.STAValue, vo.STASourceUserProvided); err != nil {
			s.logger.Warn("Failed to set STA value", map[string]interface{}{
				"photo_id": photo.ID().String(),
				"error":    err.Error(),
			})
		}
	}

	// Create pending upload record
	pendingUpload := &repository.PendingUpload{
		UploadToken: uploadToken,
		PhotoID:     photo.ID(),
		APIKeyID:    apiKeyID,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(model.UploadTokenExpiry),
		Status:      vo.UploadStatusPending,
	}

	// Generate signed URL (15 minutes expiry)
	signedURL, err := s.gcsClient.GenerateSignedURL(gcsObjectName, req.FileMetadata.ContentType, 15)
	if err != nil {
		s.logger.Error("Failed to generate signed URL", err, map[string]interface{}{
			"api_key_id":      apiKeyID,
			"gcs_object_name": gcsObjectName,
		})
		return nil, NewServiceError("STORAGE_ERROR", "Failed to generate signed URL", err)
	}

	// Save photo and pending upload
	if err := s.photoRepo.Create(ctx, photo); err != nil {
		s.logger.Error("Failed to create photo record", err, nil)
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to save photo record", err)
	}

	if err := s.pendingUploadRepo.Create(ctx, pendingUpload); err != nil {
		s.logger.Error("Failed to create pending upload record", err, nil)
		// Attempt to rollback photo creation
		_ = s.photoRepo.HardDelete(ctx, photo.ID())
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to save pending upload", err)
	}

	s.logger.Info("Upload URL generated", map[string]interface{}{
		"photo_id":        photo.ID().String(),
		"upload_token":    uploadToken.String(),
		"gcs_object_name": gcsObjectName,
		"api_key_id":      apiKeyID,
		"expires_at":      pendingUpload.ExpiresAt.Format(time.RFC3339),
	})

	return &rest.GetSignedUploadURLResponse{
		UploadToken: uploadToken,
		SignedURL:   signedURL,
		PhotoID:     photo.ID(),
		ExpiresAt:   pendingUpload.ExpiresAt,
	}, nil
}

// ConfirmUpload handles Phase 2 of the two-phase upload workflow.
// It validates the token, verifies the file exists in GCS, and marks the upload as completed.
func (s *UploadServiceImpl) ConfirmUpload(
	ctx context.Context,
	token vo.UploadToken,
	apiKeyID string,
) (*rest.ConfirmUploadResponse, error) {
	// Validate token format
	if !token.IsValid() {
		return nil, ErrInvalidToken
	}

	// Get pending upload by token
	pendingUpload, err := s.pendingUploadRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil, ErrTokenNotFound
		}
		s.logger.Error("Failed to get pending upload", err, map[string]interface{}{
			"upload_token": token.String(),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to retrieve upload record", err)
	}

	// Verify token belongs to the same API key
	if pendingUpload.APIKeyID != apiKeyID {
		s.logger.Warn("API key mismatch for upload token", map[string]interface{}{
			"upload_token":     token.String(),
			"expected_api_key": pendingUpload.APIKeyID,
			"actual_api_key":   apiKeyID,
		})
		return nil, ErrTokenNotFound
	}

	// Check token status
	switch pendingUpload.Status {
	case vo.UploadStatusCompleted:
		return nil, ErrTokenAlreadyUsed
	case vo.UploadStatusExpired:
		return nil, ErrTokenExpired
	}
	// If status is Pending, continue with the upload confirmation

	// Check if token has expired
	if time.Now().After(pendingUpload.ExpiresAt) {
		// Mark as expired
		_, _ = s.pendingUploadRepo.MarkAsExpired(ctx, time.Now())
		return nil, ErrTokenExpired
	}

	// Get the associated photo
	photo, err := s.photoRepo.GetByID(ctx, pendingUpload.PhotoID)
	if err != nil {
		if errors.Is(err, model.ErrPhotoNotFound) {
			return nil, ErrPhotoNotFound
		}
		s.logger.Error("Failed to get photo", err, map[string]interface{}{
			"photo_id": pendingUpload.PhotoID.String(),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to retrieve photo", err)
	}

	// Verify file exists in GCS
	exists, err := s.gcsClient.FileExists(photo.GCSObjectName())
	if err != nil {
		s.logger.Error("Failed to check file existence in GCS", err, map[string]interface{}{
			"gcs_object_name": photo.GCSObjectName(),
		})
		return nil, NewServiceError("STORAGE_ERROR", "Failed to verify file in storage", err)
	}

	if !exists {
		s.logger.Warn("File not found in GCS", map[string]interface{}{
			"gcs_object_name": photo.GCSObjectName(),
			"photo_id":        photo.ID().String(),
		})
		return nil, ErrFileNotFound
	}

	// Mark upload as completed
	if err := s.pendingUploadRepo.MarkAsCompleted(ctx, token); err != nil {
		s.logger.Error("Failed to mark upload as completed", err, map[string]interface{}{
			"upload_token": token.String(),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to update upload status", err)
	}

	// Log success
	s.logger.Info("Upload confirmed", map[string]interface{}{
		"photo_id":     photo.ID().String(),
		"upload_token": token.String(),
		"api_key_id":   apiKeyID,
		"gcs_object":   photo.GCSObjectName(),
	})

	return &rest.ConfirmUploadResponse{
		PhotoID: photo.ID(),
		Message: "Upload confirmed successfully",
	}, nil
}
