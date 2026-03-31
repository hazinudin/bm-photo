package service

import (
	"context"
	"time"

	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// AuditLogServiceImpl implements AuditLogService for operation logging
type AuditLogServiceImpl struct {
	auditLogRepo repository.AuditLogRepository
	logger       Logger
}

// NewAuditLogService creates a new AuditLogService instance
func NewAuditLogService(
	auditLogRepo repository.AuditLogRepository,
	logger Logger,
) *AuditLogServiceImpl {
	return &AuditLogServiceImpl{
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

// Audit operation constants
const (
	OperationUploadRequest = "upload_signed_url"
	OperationUploadConfirm = "upload_confirm"
	OperationPhotoUpdate   = "photo_update"
	OperationPhotoDelete   = "photo_delete"
	OperationPhotoRestore  = "photo_restore"
	OperationPhotoDownload = "photo_download"
)

// LogUploadRequest logs a signed URL generation request
func (s *AuditLogServiceImpl) LogUploadRequest(
	ctx context.Context,
	photoID string,
	apiKeyID string,
	details map[string]interface{},
) {
	photoIDVO := vo.MustParsePhotoID(photoID)
	entry := &repository.AuditLogEntry{
		PhotoID:    &photoIDVO,
		Operation:  OperationUploadRequest,
		APIKeyID:   apiKeyID,
		OperatedAt: time.Now(),
		Details:    details,
	}

	if err := s.auditLogRepo.Create(ctx, entry); err != nil {
		s.logger.Error("Failed to create audit log for upload request", err, map[string]interface{}{
			"photo_id":   photoID,
			"api_key_id": apiKeyID,
		})
		return
	}

	s.logger.Debug("Audit log created", map[string]interface{}{
		"operation":  OperationUploadRequest,
		"photo_id":   photoID,
		"api_key_id": apiKeyID,
	})
}

// LogUploadConfirm logs an upload confirmation
func (s *AuditLogServiceImpl) LogUploadConfirm(
	ctx context.Context,
	photoID string,
	apiKeyID string,
) {
	photoIDVO := vo.MustParsePhotoID(photoID)
	entry := &repository.AuditLogEntry{
		PhotoID:    &photoIDVO,
		Operation:  OperationUploadConfirm,
		APIKeyID:   apiKeyID,
		OperatedAt: time.Now(),
		Details: map[string]interface{}{
			"confirmed": true,
		},
	}

	if err := s.auditLogRepo.Create(ctx, entry); err != nil {
		s.logger.Error("Failed to create audit log for upload confirm", err, map[string]interface{}{
			"photo_id":   photoID,
			"api_key_id": apiKeyID,
		})
		return
	}

	s.logger.Debug("Audit log created", map[string]interface{}{
		"operation":  OperationUploadConfirm,
		"photo_id":   photoID,
		"api_key_id": apiKeyID,
	})
}

// LogPhotoUpdate logs a photo metadata update
func (s *AuditLogServiceImpl) LogPhotoUpdate(
	ctx context.Context,
	photoID string,
	apiKeyID string,
	details map[string]interface{},
) {
	photoIDVO := vo.MustParsePhotoID(photoID)
	entry := &repository.AuditLogEntry{
		PhotoID:    &photoIDVO,
		Operation:  OperationPhotoUpdate,
		APIKeyID:   apiKeyID,
		OperatedAt: time.Now(),
		Details:    details,
	}

	if err := s.auditLogRepo.Create(ctx, entry); err != nil {
		s.logger.Error("Failed to create audit log for photo update", err, map[string]interface{}{
			"photo_id":   photoID,
			"api_key_id": apiKeyID,
		})
		return
	}

	s.logger.Debug("Audit log created", map[string]interface{}{
		"operation":  OperationPhotoUpdate,
		"photo_id":   photoID,
		"api_key_id": apiKeyID,
	})
}

// LogPhotoDelete logs a photo deletion
func (s *AuditLogServiceImpl) LogPhotoDelete(
	ctx context.Context,
	photoID string,
	apiKeyID string,
	hard bool,
) {
	details := map[string]interface{}{
		"hard_delete": hard,
	}

	if hard {
		details["deletion_type"] = "hard"
	} else {
		details["deletion_type"] = "soft"
	}

	photoIDVO := vo.MustParsePhotoID(photoID)
	entry := &repository.AuditLogEntry{
		PhotoID:    &photoIDVO,
		Operation:  OperationPhotoDelete,
		APIKeyID:   apiKeyID,
		OperatedAt: time.Now(),
		Details:    details,
	}

	if err := s.auditLogRepo.Create(ctx, entry); err != nil {
		s.logger.Error("Failed to create audit log for photo delete", err, map[string]interface{}{
			"photo_id":   photoID,
			"api_key_id": apiKeyID,
			"hard":       hard,
		})
		return
	}

	s.logger.Debug("Audit log created", map[string]interface{}{
		"operation":  OperationPhotoDelete,
		"photo_id":   photoID,
		"api_key_id": apiKeyID,
		"hard":       hard,
	})
}

// LogPhotoRestore logs a photo restore operation (for soft-deleted photos)
func (s *AuditLogServiceImpl) LogPhotoRestore(
	ctx context.Context,
	photoID string,
	apiKeyID string,
) {
	photoIDVO := vo.MustParsePhotoID(photoID)
	entry := &repository.AuditLogEntry{
		PhotoID:    &photoIDVO,
		Operation:  OperationPhotoRestore,
		APIKeyID:   apiKeyID,
		OperatedAt: time.Now(),
		Details: map[string]interface{}{
			"restored": true,
		},
	}

	if err := s.auditLogRepo.Create(ctx, entry); err != nil {
		s.logger.Error("Failed to create audit log for photo restore", err, map[string]interface{}{
			"photo_id":   photoID,
			"api_key_id": apiKeyID,
		})
		return
	}

	s.logger.Debug("Audit log created", map[string]interface{}{
		"operation":  OperationPhotoRestore,
		"photo_id":   photoID,
		"api_key_id": apiKeyID,
	})
}

// LogPhotoDownload logs a photo download operation
func (s *AuditLogServiceImpl) LogPhotoDownload(
	ctx context.Context,
	photoID string,
	apiKeyID string,
) {
	photoIDVO := vo.MustParsePhotoID(photoID)
	entry := &repository.AuditLogEntry{
		PhotoID:    &photoIDVO,
		Operation:  OperationPhotoDownload,
		APIKeyID:   apiKeyID,
		OperatedAt: time.Now(),
		Details: map[string]interface{}{
			"downloaded": true,
		},
	}

	if err := s.auditLogRepo.Create(ctx, entry); err != nil {
		s.logger.Error("Failed to create audit log for photo download", err, map[string]interface{}{
			"photo_id":   photoID,
			"api_key_id": apiKeyID,
		})
		return
	}

	s.logger.Debug("Audit log created", map[string]interface{}{
		"operation":  OperationPhotoDownload,
		"photo_id":   photoID,
		"api_key_id": apiKeyID,
	})
}

// NullAuditLogService is a no-op audit log service for testing
type NullAuditLogService struct{}

// NewNullAuditLogService creates a null audit log service
func NewNullAuditLogService() *NullAuditLogService {
	return &NullAuditLogService{}
}

// LogUploadRequest is a no-op
func (s *NullAuditLogService) LogUploadRequest(ctx context.Context, photoID, apiKeyID string, details map[string]interface{}) {
}

// LogUploadConfirm is a no-op
func (s *NullAuditLogService) LogUploadConfirm(ctx context.Context, photoID, apiKeyID string) {
}

// LogPhotoUpdate is a no-op
func (s *NullAuditLogService) LogPhotoUpdate(ctx context.Context, photoID, apiKeyID string, details map[string]interface{}) {
}

// LogPhotoDelete is a no-op
func (s *NullAuditLogService) LogPhotoDelete(ctx context.Context, photoID, apiKeyID string, hard bool) {
}
