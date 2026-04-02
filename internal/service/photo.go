package service

import (
	"context"
	"errors"
	"time"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// PhotoServiceImpl implements PhotoService for photo CRUD and browsing
type PhotoServiceImpl struct {
	photoRepo         repository.PhotoRepository
	pendingUploadRepo repository.PendingUploadRepository
	gcsClient         GCSClient
	logger            Logger
	auditSvc          AuditLogService
}

// NewPhotoService creates a new PhotoService instance
func NewPhotoService(
	photoRepo repository.PhotoRepository,
	pendingUploadRepo repository.PendingUploadRepository,
	gcsClient GCSClient,
	logger Logger,
	auditSvc AuditLogService,
) *PhotoServiceImpl {
	return &PhotoServiceImpl{
		photoRepo:         photoRepo,
		pendingUploadRepo: pendingUploadRepo,
		gcsClient:         gcsClient,
		logger:            logger,
		auditSvc:          auditSvc,
	}
}

// GetByID retrieves a photo by its ID
func (s *PhotoServiceImpl) GetByID(ctx context.Context, id vo.PhotoID) (*entity.Photo, error) {
	if !id.IsValid() {
		return nil, ErrPhotoNotFound
	}

	photo, err := s.photoRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, model.ErrPhotoNotFound) {
			return nil, ErrPhotoNotFound
		}
		s.logger.Error("Failed to get photo", err, map[string]interface{}{
			"photo_id": id.String(),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to retrieve photo", err)
	}

	// Check if photo is deleted
	if photo.IsDeleted() {
		return nil, ErrPhotoDeleted
	}

	return photo, nil
}

// Browse retrieves photos with optional filters (route, STA, lane)
func (s *PhotoServiceImpl) Browse(ctx context.Context, filter repository.BrowseFilter) (*rest.BrowsePhotosResponse, error) {
	// Validate and set defaults
	if filter.Page <= 0 {
		filter.Page = model.DefaultPage
	}
	if filter.PerPage <= 0 {
		filter.PerPage = model.DefaultPerPage
	}
	if filter.PerPage > model.MaxPerPage {
		filter.PerPage = model.MaxPerPage
	}

	// Validate STA range if provided
	if filter.STAStart != nil && filter.STAEnd != nil {
		if *filter.STAStart > *filter.STAEnd {
			return nil, ErrInvalidSTAValue
		}
	}

	result, err := s.photoRepo.Browse(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to browse photos", err, map[string]interface{}{
			"route_id": filter.RouteID,
			"page":     filter.Page,
			"per_page": filter.PerPage,
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to browse photos", err)
	}

	// Convert to response DTOs
	photos := make([]rest.PhotoSummary, 0, len(result.Photos))
	for _, photo := range result.Photos {
		// Skip deleted photos
		if photo.IsDeleted() {
			continue
		}

		photos = append(photos, rest.PhotoSummary{
			PhotoID:          photo.ID(),
			RouteID:          photo.RouteID(),
			LaneCode:         photo.LaneCode(),
			STAValue:         photo.STAValue(),
			GCSURL:           s.gcsClient.GetPublicURL(photo.GCSObjectName()),
			UploadedAt:       photo.UploadedAt(),
			OriginalFileName: *photo.OriginalFilename(),
		})
	}

	// Calculate total pages
	totalPages := int(result.TotalCount) / result.PerPage
	if int(result.TotalCount)%result.PerPage > 0 {
		totalPages++
	}

	return &rest.BrowsePhotosResponse{
		Photos: photos,
		Pagination: rest.Pagination{
			CurrentPage: result.Page,
			PerPage:     result.PerPage,
			TotalCount:  result.TotalCount,
			TotalPages:  totalPages,
		},
	}, nil
}

// Search performs advanced search with multiple filters
func (s *PhotoServiceImpl) Search(ctx context.Context, filter repository.SearchFilter) (*rest.SearchPhotosResponse, error) {
	// Validate and set defaults
	if filter.Page <= 0 {
		filter.Page = model.DefaultPage
	}
	if filter.PerPage <= 0 {
		filter.PerPage = model.DefaultPerPage
	}
	if filter.PerPage > model.MaxPerPage {
		filter.PerPage = model.MaxPerPage
	}

	result, err := s.photoRepo.Search(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to search photos", err, map[string]interface{}{
			"route_ids": filter.RouteIDs,
			"page":      filter.Page,
			"per_page":  filter.PerPage,
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to search photos", err)
	}

	// Convert to response DTOs
	photos := make([]rest.PhotoSummary, 0, len(result.Photos))
	for _, photo := range result.Photos {
		// Skip deleted photos
		if photo.IsDeleted() {
			continue
		}

		photos = append(photos, rest.PhotoSummary{
			PhotoID:    photo.ID(),
			RouteID:    photo.RouteID(),
			LaneCode:   photo.LaneCode(),
			STAValue:   photo.STAValue(),
			GCSURL:     s.gcsClient.GetPublicURL(photo.GCSObjectName()),
			UploadedAt: photo.UploadedAt(),
		})
	}

	// Calculate total pages
	totalPages := int(result.TotalCount) / result.PerPage
	if int(result.TotalCount)%result.PerPage > 0 {
		totalPages++
	}

	return &rest.SearchPhotosResponse{
		Photos: photos,
		Pagination: rest.Pagination{
			CurrentPage: result.Page,
			PerPage:     result.PerPage,
			TotalCount:  result.TotalCount,
			TotalPages:  totalPages,
		},
	}, nil
}

// Update modifies photo metadata (description, tags, lane)
func (s *PhotoServiceImpl) Update(
	ctx context.Context,
	id vo.PhotoID,
	req *rest.UpdatePhotoRequest,
) (*rest.UpdatePhotoResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Get photo
	photo, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Description != nil {
		if err := photo.UpdateDescription(*req.Description); err != nil {
			return nil, err
		}
	}

	if req.Tags != nil {
		if err := photo.UpdateTags(req.Tags); err != nil {
			return nil, err
		}
	}

	if req.LaneCode != nil {
		if err := photo.UpdateLaneCode(*req.LaneCode); err != nil {
			return nil, err
		}
	}

	// Save to repository
	if err := s.photoRepo.Update(ctx, photo); err != nil {
		s.logger.Error("Failed to update photo", err, map[string]interface{}{
			"photo_id": id.String(),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to update photo", err)
	}

	// Log audit
	s.auditSvc.LogPhotoUpdate(ctx, id.String(), photo.UploadedBy(), map[string]interface{}{
		"description": req.Description,
		"tags":        req.Tags,
		"lane_code":   req.LaneCode,
	})

	s.logger.Info("Photo updated", map[string]interface{}{
		"photo_id": id.String(),
	})

	return &rest.UpdatePhotoResponse{
		PhotoID:     photo.ID(),
		Description: photo.Description(),
		Tags:        photo.Tags(),
		LaneCode:    photo.LaneCode(),
		UpdatedAt:   photo.UpdatedAt(),
	}, nil
}

// Delete soft-deletes or hard-deletes a photo
func (s *PhotoServiceImpl) Delete(
	ctx context.Context,
	id vo.PhotoID,
	hard bool,
	apiKeyID string,
) (*rest.DeletePhotoResponse, error) {
	// Get photo - use GetByIDIncludeDeleted for hard delete to find soft-deleted photos
	var photo *entity.Photo
	var err error

	if hard {
		photo, err = s.photoRepo.GetByIDIncludeDeleted(ctx, id)
	} else {
		photo, err = s.photoRepo.GetByID(ctx, id)
	}

	if err != nil {
		if errors.Is(err, model.ErrPhotoNotFound) {
			return nil, ErrPhotoNotFound
		}
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to retrieve photo", err)
	}

	// Check if already deleted (for soft delete)
	if photo.IsDeleted() && !hard {
		return nil, ErrPhotoDeleted
	}

	var deletedAt time.Time

	if hard {
		// Hard delete - remove from database and GCS
		gcsObjectName := photo.GCSObjectName()

		// Delete from GCS
		if err := s.gcsClient.DeleteFile(gcsObjectName); err != nil {
			s.logger.Warn("Failed to delete file from GCS", map[string]interface{}{
				"gcs_object_name": gcsObjectName,
				"error":           err.Error(),
			})
			// Continue with database deletion even if GCS fails
			// File may already be deleted or unreachable
		}

		// Delete associated pending uploads first (due to FK constraint)
		if err := s.pendingUploadRepo.DeleteByPhotoID(ctx, id); err != nil {
			s.logger.Warn("Failed to delete pending uploads", map[string]interface{}{
				"photo_id": id.String(),
				"error":    err.Error(),
			})
			// Continue with photo deletion - pending uploads may already be gone
		}

		// Hard delete from database
		if err := s.photoRepo.HardDelete(ctx, id); err != nil {
			s.logger.Error("Failed to hard delete photo", err, map[string]interface{}{
				"photo_id": id.String(),
			})
			return nil, NewServiceError("INTERNAL_ERROR", "Failed to delete photo", err)
		}

		deletedAt = time.Now()
	} else {
		// Soft delete
		if err := photo.SoftDelete(apiKeyID); err != nil {
			if errors.Is(err, entity.ErrPhotoDeleted) {
				return nil, ErrPhotoDeleted
			}
			return nil, err
		}

		if err := s.photoRepo.SoftDelete(ctx, id, apiKeyID); err != nil {
			s.logger.Error("Failed to soft delete photo", err, map[string]interface{}{
				"photo_id": id.String(),
			})
			return nil, NewServiceError("INTERNAL_ERROR", "Failed to delete photo", err)
		}

		deletedAt = *photo.DeletedAt()
	}

	// Log audit
	s.auditSvc.LogPhotoDelete(ctx, id.String(), apiKeyID, hard)

	deletionType := "soft"
	if hard {
		deletionType = "hard"
	}

	s.logger.Info("Photo deleted", map[string]interface{}{
		"photo_id":      id.String(),
		"deletion_type": deletionType,
		"api_key_id":    apiKeyID,
	})

	return &rest.DeletePhotoResponse{
		PhotoID:      id,
		DeletedAt:    deletedAt,
		DeletionType: deletionType,
	}, nil
}

// BuildPhotoResponse builds a complete photo response DTO from an entity
func BuildPhotoResponse(photo *entity.Photo, downloadURL string) *rest.PhotoResponse {
	resp := &rest.PhotoResponse{
		PhotoID:       photo.ID(),
		RouteID:       photo.RouteID(),
		LaneCode:      photo.LaneCode(),
		Latitude:      photo.Latitude(),
		Longitude:     photo.Longitude(),
		STAValue:      photo.STAValue(),
		STASource:     photo.STASource(),
		FileFormat:    photo.FileFormat(),
		FileSizeBytes: photo.FileSizeBytes(),
		Description:   photo.Description(),
		Tags:          photo.Tags(),
		UploadedAt:    photo.UploadedAt(),
		DownloadURL:   downloadURL,
	}

	return resp
}

// GenerateDownloadURL generates a signed download URL for a photo
func GenerateDownloadURL(gcsClient GCSClient, objectName string, expiryMinutes int) (string, error) {
	if gcsClient == nil {
		return "", errors.New("GCS client not configured")
	}
	return gcsClient.GenerateSignedURL(objectName, "image/jpeg", expiryMinutes)
}
