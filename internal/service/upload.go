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
// photos/{year}/{route_id}/{route_id}_{year}_{lane}_{photoID}.{ext}
func generateGCSObjectName(routeID, laneCode, photoID, contentType string, year int) string {
	// Derive extension from content type
	ext := "jpg"
	if contentType == "image/png" {
		ext = "png"
	}
	return fmt.Sprintf("photos/%d/%s/%s_%d_%s_%s.%s", year, routeID, routeID, year, laneCode, photoID, ext)
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

	// Extract survey year - default to current year if not provided
	surveyYear := time.Now().Year()
	if req.PhotoAttributes.SurveyYear != nil {
		surveyYear = *req.PhotoAttributes.SurveyYear
	}

	// Extract route ID and lane code
	// Route ID and lane may be empty initially, will be updated in confirm phase
	routeID := req.PhotoAttributes.RouteID
	laneCode := req.PhotoAttributes.LaneCode
	if laneCode == "" {
		laneCode = "unknown"
	}

	// Create photo entity with pending status (use placeholder GCS object name for now)
	photoParams := entity.PhotoParams{
		RouteID:          routeID,
		LaneCode:         laneCode,
		SurveyYear:       surveyYear,
		GCSObjectName:    "placeholder", // Will be updated with real ID below
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

	// Generate the correct GCS object name using the real photo ID
	gcsObjectName := generateGCSObjectName(routeID, laneCode, photo.ID().String(), req.FileMetadata.ContentType, surveyYear)
	if err := photo.SetGCSObjectName(gcsObjectName); err != nil {
		s.logger.Error("Failed to set GCS object name", err, map[string]interface{}{
			"photo_id": photo.ID().String(),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to set GCS object name", err)
	}

	// Set coordinates if provided
	if req.PhotoAttributes.Latitude != nil && req.PhotoAttributes.Longitude != nil {
		if err := photo.SetCoordinates(*req.PhotoAttributes.Latitude, *req.PhotoAttributes.Longitude); err != nil {
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

	// Update photo upload status to completed
	if err := s.photoRepo.UpdateUploadStatus(ctx, photo.ID(), vo.UploadStatusCompleted); err != nil {
		s.logger.Error("Failed to update photo upload status", err, map[string]interface{}{
			"photo_id": photo.ID().String(),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to update photo status", err)
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

// GetNewSignedURL generates a new signed URL for a pending photo upload.
// This is used when the original upload token has expired or the upload failed.
// It validates that the photo exists, is in pending status, belongs to the requesting API key,
// and has not exceeded the retry limit. Old pending tokens are invalidated and a new one is created.
func (s *UploadServiceImpl) GetNewSignedURL(
	ctx context.Context,
	photoID string,
	apiKeyID string,
) (*rest.GetNewSignedURLResponse, error) {
	// Parse and validate photo ID
	photoIDVo, err := vo.ParsePhotoID(photoID)
	if err != nil {
		s.logger.Warn("Invalid photo ID format", map[string]interface{}{
			"photo_id": photoID,
			"error":    err.Error(),
		})
		return nil, ErrPhotoNotFound
	}

	// 1. Get photo by ID
	photo, err := s.photoRepo.GetByID(ctx, photoIDVo)
	if err != nil {
		if errors.Is(err, model.ErrPhotoNotFound) {
			return nil, ErrPhotoNotFound
		}
		s.logger.Error("Failed to get photo", err, map[string]interface{}{
			"photo_id": photoID,
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to retrieve photo", err)
	}

	// 2. Verify photo exists and status is 'pending'
	if photo.IsDeleted() {
		return nil, ErrPhotoDeleted
	}
	if !photo.IsUploadPending() {
		return nil, ErrUploadNotPending
	}

	// 3. Verify photo.uploaded_by_api_key == apiKeyID (ownership check)
	if err := photo.VerifyOwnership(apiKeyID); err != nil {
		s.logger.Warn("Photo ownership verification failed", map[string]interface{}{
			"photo_id":   photoID,
			"api_key_id": apiKeyID,
			"owner_id":   photo.UploadedBy(),
			"error":      err.Error(),
		})
		return nil, ErrPhotoNotOwned
	}

	// 4. Check retry_count < MaxRetriesPerPhoto
	if !photo.CanRetryUpload() {
		s.logger.Warn("Retry limit exceeded", map[string]interface{}{
			"photo_id":    photoID,
			"retry_count": photo.RetryCount(),
			"max_retries": model.MaxRetriesPerPhoto,
		})
		return nil, ErrRetryLimitExceeded
	}

	// 5. Mark existing pending_uploads for this photo as 'expired'
	if err := s.pendingUploadRepo.ExpireTokensByPhotoID(ctx, photoIDVo); err != nil {
		s.logger.Error("Failed to expire old tokens", err, map[string]interface{}{
			"photo_id": photoID,
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to invalidate old tokens", err)
	}

	// 6. Generate new upload_token
	newUploadToken := vo.NewUploadToken()

	// 7. Generate new signed URL
	signedURL, err := s.gcsClient.GenerateSignedURL(photo.GCSObjectName(), photo.FileFormat().ContentType(), 15)
	if err != nil {
		s.logger.Error("Failed to generate signed URL", err, map[string]interface{}{
			"photo_id":        photoID,
			"gcs_object_name": photo.GCSObjectName(),
		})
		return nil, NewServiceError("STORAGE_ERROR", "Failed to generate signed URL", err)
	}

	// 8. Create new pending_upload record
	expiresAt := time.Now().Add(model.UploadTokenExpiry)
	newPendingUpload := &repository.PendingUpload{
		UploadToken: newUploadToken,
		PhotoID:     photoIDVo,
		APIKeyID:    apiKeyID,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Status:      vo.UploadStatusPending,
	}

	if err := s.pendingUploadRepo.Create(ctx, newPendingUpload); err != nil {
		s.logger.Error("Failed to create pending upload record", err, map[string]interface{}{
			"photo_id": photoID,
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to save pending upload", err)
	}

	// 9. Increment retry_count on photo
	if err := s.photoRepo.IncrementRetryCount(ctx, photoIDVo); err != nil {
		s.logger.Error("Failed to increment retry count", err, map[string]interface{}{
			"photo_id": photoID,
		})
		// Continue anyway - the upload can still proceed
	}

	// Get updated retry count
	newRetryCount := photo.RetryCount() + 1

	s.logger.Info("New signed URL generated for retry", map[string]interface{}{
		"photo_id":     photoID,
		"upload_token": newUploadToken.String(),
		"retry_count":  newRetryCount,
		"max_retries":  model.MaxRetriesPerPhoto,
		"expires_at":   expiresAt.Format(time.RFC3339),
	})

	return &rest.GetNewSignedURLResponse{
		PhotoID:     photoIDVo,
		UploadToken: newUploadToken,
		SignedURL:   signedURL,
		ExpiresAt:   expiresAt,
		RetryCount:  newRetryCount,
		MaxRetries:  model.MaxRetriesPerPhoto,
	}, nil
}
