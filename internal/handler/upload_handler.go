package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/service"
)

// UploadHandler handles upload-related HTTP endpoints for two-phase upload workflow
type UploadHandler struct {
	*BaseHandler
	uploadSvc service.UploadService
}

// NewUploadHandler creates a new UploadHandler instance
func NewUploadHandler(uploadSvc service.UploadService, logger *slog.Logger) *UploadHandler {
	return &UploadHandler{
		BaseHandler: NewBaseHandler(logger),
		uploadSvc:   uploadSvc,
	}
}

// GetSignedUploadURL handles Phase 1 of the two-phase upload workflow.
// POST /api/v1/uploads/signed-url
func (h *UploadHandler) GetSignedUploadURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	apiKeyID := GetAPIKeyID(ctx)
	if apiKeyID == "" {
		h.writeError(w, http.StatusUnauthorized, "API key ID not found in context", "UNAUTHORIZED")
		return
	}

	var req rest.GetSignedUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	resp, err := h.uploadSvc.GetSignedURL(ctx, &req, apiKeyID)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, resp)
}

// ConfirmUpload handles Phase 2 of the two-phase upload workflow.
// POST /api/v1/uploads/confirm
func (h *UploadHandler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	apiKeyID := GetAPIKeyID(ctx)
	if apiKeyID == "" {
		h.writeError(w, http.StatusUnauthorized, "API key ID not found in context", "UNAUTHORIZED")
		return
	}

	var req rest.ConfirmUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	// Validate the request
	if err := req.Validate(); err != nil {
		h.handleValidationError(w, err)
		return
	}

	resp, err := h.uploadSvc.ConfirmUpload(ctx, req.UploadToken, apiKeyID)
	if err != nil {
		h.handleUploadError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// handleUploadError maps service errors to appropriate HTTP status codes
func (h *UploadHandler) handleUploadError(w http.ResponseWriter, err error) {
	// Check if it's a model.ValidationError
	var ve model.ValidationError
	if errors.As(err, &ve) {
		h.handleValidationError(w, err)
		return
	}

	// Map service errors to HTTP status codes
	switch {
	case errors.Is(err, service.ErrUploadQuotaExceeded):
		h.writeError(w, http.StatusTooManyRequests, err.Error(), "QUOTA_EXCEEDED")
	case errors.Is(err, service.ErrUnsupportedFormat):
		h.writeError(w, http.StatusBadRequest, err.Error(), "UNSUPPORTED_FORMAT")
	case errors.Is(err, service.ErrFileTooLarge):
		h.writeError(w, http.StatusRequestEntityTooLarge, err.Error(), "FILE_TOO_LARGE")
	case errors.Is(err, service.ErrInvalidToken):
		h.writeError(w, http.StatusBadRequest, err.Error(), "INVALID_TOKEN")
	case errors.Is(err, service.ErrTokenNotFound):
		h.writeError(w, http.StatusNotFound, err.Error(), "TOKEN_NOT_FOUND")
	case errors.Is(err, service.ErrTokenAlreadyUsed):
		h.writeError(w, http.StatusConflict, err.Error(), "TOKEN_ALREADY_USED")
	case errors.Is(err, service.ErrTokenExpired):
		h.writeError(w, http.StatusGone, err.Error(), "TOKEN_EXPIRED")
	case errors.Is(err, service.ErrFileNotFound):
		h.writeError(w, http.StatusNotFound, err.Error(), "FILE_NOT_FOUND")
	case errors.Is(err, service.ErrPhotoNotFound):
		h.writeError(w, http.StatusNotFound, err.Error(), "PHOTO_NOT_FOUND")
	case service.IsServiceError(err):
		h.handleServiceError(w, err)
	default:
		h.handleServiceError(w, err)
	}
}
