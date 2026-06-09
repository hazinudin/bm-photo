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
			SurveyYear:       photo.SurveyYear(),
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

// GetStats returns photo count statistics grouped by lane
func (s *PhotoServiceImpl) GetStats(ctx context.Context, filter repository.StatsFilter) (*rest.PhotoStatsResponse, error) {
	result, err := s.photoRepo.GetStats(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get photo stats", err, map[string]interface{}{
			"route_id": filter.RouteID,
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to get photo stats", err)
	}

	laneStats := make([]rest.LaneStats, 0, len(result.LaneStats))
	for _, ls := range result.LaneStats {
		laneStats = append(laneStats, rest.LaneStats{
			LaneCode: ls.LaneCode,
			Count:    ls.Count,
		})
	}

	return &rest.PhotoStatsResponse{
		RouteID:    result.RouteID,
		TotalCount: result.TotalCount,
		LaneStats:  laneStats,
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
			SurveyYear: photo.SurveyYear(),
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

	if req.Latitude != nil && req.Longitude != nil {
		if err := photo.SetCoordinates(*req.Latitude, *req.Longitude); err != nil {
			return nil, err
		}
	}

	if req.STAValue != nil {
		if err := photo.SetSTA(*req.STAValue, vo.STASourceUserProvided); err != nil {
			return nil, err
		}
	}

	if req.SurveyYear != nil {
		if err := photo.SetSurveyYear(*req.SurveyYear); err != nil {
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
		"latitude":    req.Latitude,
		"longitude":   req.Longitude,
		"sta_value":   req.STAValue,
		"survey_year": req.SurveyYear,
	})

	s.logger.Info("Photo updated", map[string]interface{}{
		"photo_id": id.String(),
	})

	return &rest.UpdatePhotoResponse{
		PhotoID:     photo.ID(),
		Description: photo.Description(),
		Tags:        photo.Tags(),
		LaneCode:    photo.LaneCode(),
		Latitude:    photo.Latitude(),
		Longitude:   photo.Longitude(),
		STAValue:    photo.STAValue(),
		STASource:   photo.STASource(),
		UpdatedAt:   photo.UpdatedAt(),
		SurveyYear:  photo.SurveyYear(),
	}, nil
}

// BatchUpdate modifies metadata for multiple photos in a single operation
func (s *PhotoServiceImpl) BatchUpdate(
	ctx context.Context,
	req *rest.BatchUpdateRequest,
) (*rest.BatchUpdateResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	photoIDs := make([]vo.PhotoID, 0, len(req.Updates))
	for _, item := range req.Updates {
		id, err := vo.ParsePhotoID(item.PhotoID)
		if err != nil {
			return nil, model.NewValidationError("photo_id", "invalid photo_id format: "+item.PhotoID)
		}
		photoIDs = append(photoIDs, id)
	}

	photos, err := s.photoRepo.GetByIDs(ctx, photoIDs)
	if err != nil {
		s.logger.Error("Failed to batch fetch photos", err, map[string]interface{}{
			"count": len(photoIDs),
		})
		return nil, NewServiceError("INTERNAL_ERROR", "Failed to fetch photos", err)
	}

	photoMap := make(map[string]*entity.Photo, len(photos))
	for _, photo := range photos {
		photoMap[photo.ID().String()] = photo
	}

	response := &rest.BatchUpdateResponse{
		Total:   len(req.Updates),
		Results: make([]rest.BatchUpdateItemResult, 0, len(req.Updates)),
	}

	for _, item := range req.Updates {
		result := rest.BatchUpdateItemResult{
			PhotoID: item.PhotoID,
		}

		photo, exists := photoMap[item.PhotoID]
		if !exists {
			result.Status = "error"
			result.Error = "photo not found or has been deleted"
			result.ErrorCode = "PHOTO_NOT_FOUND"
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		updateReq := &rest.UpdatePhotoRequest{
			Description: item.Description,
			Tags:        item.Tags,
			SurveyYear:  item.SurveyYear,
			LaneCode:    item.LaneCode,
			Latitude:    item.Latitude,
			Longitude:   item.Longitude,
			STAValue:    item.STAValue,
		}

		if err := updateReq.Validate(); err != nil {
			result.Status = "error"
			result.Error = err.Error()
			result.ErrorCode = "VALIDATION_ERROR"
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		if updateReq.Description != nil {
			if err := photo.UpdateDescription(*updateReq.Description); err != nil {
				result.Status = "error"
				result.Error = err.Error()
				result.ErrorCode = "VALIDATION_ERROR"
				response.Failed++
				response.Results = append(response.Results, result)
				continue
			}
		}

		if updateReq.Tags != nil {
			if err := photo.UpdateTags(updateReq.Tags); err != nil {
				result.Status = "error"
				result.Error = err.Error()
				result.ErrorCode = "VALIDATION_ERROR"
				response.Failed++
				response.Results = append(response.Results, result)
				continue
			}
		}

		if updateReq.LaneCode != nil {
			if err := photo.UpdateLaneCode(*updateReq.LaneCode); err != nil {
				result.Status = "error"
				result.Error = err.Error()
				result.ErrorCode = "VALIDATION_ERROR"
				response.Failed++
				response.Results = append(response.Results, result)
				continue
			}
		}

		if updateReq.Latitude != nil && updateReq.Longitude != nil {
			if err := photo.SetCoordinates(*updateReq.Latitude, *updateReq.Longitude); err != nil {
				result.Status = "error"
				result.Error = err.Error()
				result.ErrorCode = "VALIDATION_ERROR"
				response.Failed++
				response.Results = append(response.Results, result)
				continue
			}
		}

		if updateReq.STAValue != nil {
			if err := photo.SetSTA(*updateReq.STAValue, vo.STASourceUserProvided); err != nil {
				result.Status = "error"
				result.Error = err.Error()
				result.ErrorCode = "VALIDATION_ERROR"
				response.Failed++
				response.Results = append(response.Results, result)
				continue
			}
		}

		if updateReq.SurveyYear != nil {
			if err := photo.SetSurveyYear(*updateReq.SurveyYear); err != nil {
				result.Status = "error"
				result.Error = err.Error()
				result.ErrorCode = "VALIDATION_ERROR"
				response.Failed++
				response.Results = append(response.Results, result)
				continue
			}
		}

		if err := s.photoRepo.Update(ctx, photo); err != nil {
			s.logger.Error("Failed to update photo in batch", err, map[string]interface{}{
				"photo_id": item.PhotoID,
			})
			result.Status = "error"
			result.Error = "failed to persist update"
			result.ErrorCode = "INTERNAL_ERROR"
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		result.Status = "success"
		result.Photo = &rest.UpdatePhotoResponse{
			PhotoID:     photo.ID(),
			Description: photo.Description(),
			Tags:        photo.Tags(),
			LaneCode:    photo.LaneCode(),
			Latitude:    photo.Latitude(),
			Longitude:   photo.Longitude(),
			STAValue:    photo.STAValue(),
			STASource:   photo.STASource(),
			UpdatedAt:   photo.UpdatedAt(),
			SurveyYear:  photo.SurveyYear(),
		}
		response.Succeeded++
		response.Results = append(response.Results, result)
	}

	s.auditSvc.LogPhotoUpdate(ctx, "batch", "system", map[string]interface{}{
		"total":     response.Total,
		"succeeded": response.Succeeded,
		"failed":    response.Failed,
	})

	s.logger.Info("Batch update completed", map[string]interface{}{
		"total":     response.Total,
		"succeeded": response.Succeeded,
		"failed":    response.Failed,
	})

	return response, nil
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
		PhotoID:          photo.ID(),
		RouteID:          photo.RouteID(),
		LaneCode:         photo.LaneCode(),
		Latitude:         photo.Latitude(),
		Longitude:        photo.Longitude(),
		STAValue:         photo.STAValue(),
		STASource:        photo.STASource(),
		FileFormat:       photo.FileFormat(),
		FileSizeBytes:    photo.FileSizeBytes(),
		OriginalFileName: photo.OriginalFilename(),
		Description:      photo.Description(),
		Tags:             photo.Tags(),
		UploadedAt:       photo.UploadedAt(),
		DownloadURL:      downloadURL,
		SurveyYear:       photo.SurveyYear(),
	}

	return resp
}

// GenerateDownloadURL generates a signed download URL for a photo.
// Returns service.ErrFileNotFound if the object does not exist in GCS.
func GenerateDownloadURL(gcsClient GCSClient, objectName string, expiryMinutes int) (string, error) {
	if gcsClient == nil {
		return "", errors.New("GCS client not configured")
	}
	url, err := gcsClient.GenerateDownloadURL(objectName, expiryMinutes)
	if err != nil {
		if err == ErrFileNotFound {
			return "", ErrFileNotFound
		}
		return "", err
	}
	return url, nil
}
