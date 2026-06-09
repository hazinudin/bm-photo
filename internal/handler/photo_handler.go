package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bina-marga/survey-photo/internal/model"
	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
	"github.com/bina-marga/survey-photo/internal/service"
)

const (
	downloadURLExpiryMinutes = 60
)

type PhotoHandler struct {
	*BaseHandler
	photoSvc  service.PhotoService
	gcsClient service.GCSClient
}

func NewPhotoHandler(photoSvc service.PhotoService, gcsClient service.GCSClient, logger *slog.Logger) *PhotoHandler {
	return &PhotoHandler{
		BaseHandler: NewBaseHandler(logger),
		photoSvc:    photoSvc,
		gcsClient:   gcsClient,
	}
}

// GetPhoto handles GET /api/v1/photos/{photo_id}
func (h *PhotoHandler) GetPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	photoIDStr := r.PathValue("photo_id")
	if photoIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "photo_id is required", "BAD_REQUEST")
		return
	}

	photoID, err := vo.ParsePhotoID(photoIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid photo_id format", "INVALID_PHOTO_ID")
		return
	}

	photo, err := h.photoSvc.GetByID(ctx, photoID)
	if err != nil {
		h.handlePhotoError(w, err)
		return
	}

	downloadURL, err := service.GenerateDownloadURL(h.gcsClient, photo.GCSObjectName(), downloadURLExpiryMinutes)
	if err != nil {
		h.handlePhotoError(w, err)
		return
	}

	resp := service.BuildPhotoResponse(photo, downloadURL)
	h.writeJSON(w, http.StatusOK, resp)
}

// BrowsePhotos handles GET /api/v1/photos
func (h *PhotoHandler) BrowsePhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := repository.BrowseFilter{
		Page:    ParseQueryInt(r, "page", model.DefaultPage),
		PerPage: ParseQueryInt(r, "per_page", model.DefaultPerPage),
	}

	if routeID := ParseQueryString(r, "route_id"); routeID != nil {
		filter.RouteID = *routeID
	}
	if staStart := ParseQueryFloat64(r, "sta_start"); staStart != nil {
		filter.STAStart = staStart
	}
	if staEnd := ParseQueryFloat64(r, "sta_end"); staEnd != nil {
		filter.STAEnd = staEnd
	}
	if lane := ParseQueryString(r, "lane_code"); lane != nil {
		filter.Lane = lane
	}
	if uploadedOnly := ParseQueryBool(r, "uploaded_only"); uploadedOnly != nil {
		filter.UploadedOnly = uploadedOnly
	}
	if surveyYear := ParseQueryInt(r, "survey_year", 0); surveyYear != 0 {
		filter.SurveyYear = &surveyYear
	}
	if fileName := ParseQueryString(r, "file_name"); fileName != nil {
		filter.Filename = fileName
	}

	resp, err := h.photoSvc.Browse(ctx, filter)
	if err != nil {
		if ve, ok := err.(*model.ValidationError); ok {
			h.writeErrorWithDetails(w, http.StatusBadRequest, ve.Message, "VALIDATION_ERROR", ve.Field)
			return
		}
		if se, ok := err.(*service.ServiceError); ok {
			h.handleServiceError(w, se)
			return
		}
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// GetPhotoStats handles GET /api/v1/photos/stats
func (h *PhotoHandler) GetPhotoStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := repository.StatsFilter{}

	if routeID := ParseQueryString(r, "route_id"); routeID != nil {
		filter.RouteID = *routeID
	} else {
		h.writeError(w, http.StatusBadRequest, "route_id is required", "VALIDATION_ERROR")
		return
	}
	if surveyYear := ParseQueryInt(r, "survey_year", 0); surveyYear != 0 {
		filter.SurveyYear = &surveyYear
	}
	if uploadedOnly := ParseQueryBool(r, "uploaded_only"); uploadedOnly != nil {
		filter.UploadedOnly = uploadedOnly
	}

	resp, err := h.photoSvc.GetStats(ctx, filter)
	if err != nil {
		if se, ok := err.(*service.ServiceError); ok {
			h.handleServiceError(w, se)
			return
		}
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// UpdatePhoto handles PATCH /api/v1/photos/{photo_id}
func (h *PhotoHandler) UpdatePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	photoIDStr := r.PathValue("photo_id")
	if photoIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "photo_id is required", "BAD_REQUEST")
		return
	}

	photoID, err := vo.ParsePhotoID(photoIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid photo_id format", "INVALID_PHOTO_ID")
		return
	}

	var req rest.UpdatePhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	resp, err := h.photoSvc.Update(ctx, photoID, &req)
	if err != nil {
		h.handlePhotoError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// BatchUpdatePhotos handles PATCH /api/v1/photos/batch
func (h *PhotoHandler) BatchUpdatePhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req rest.BatchUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	resp, err := h.photoSvc.BatchUpdate(ctx, &req)
	if err != nil {
		if ve, ok := err.(*model.ValidationError); ok {
			h.writeErrorWithDetails(w, http.StatusBadRequest, ve.Message, "VALIDATION_ERROR", ve.Field)
			return
		}
		if se, ok := err.(*service.ServiceError); ok {
			h.handleServiceError(w, se)
			return
		}
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// DeletePhoto handles DELETE /api/v1/photos/{photo_id}
func (h *PhotoHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	photoIDStr := r.PathValue("photo_id")
	if photoIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "photo_id is required", "BAD_REQUEST")
		return
	}

	photoID, err := vo.ParsePhotoID(photoIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid photo_id format", "INVALID_PHOTO_ID")
		return
	}

	apiKeyID := GetAPIKeyID(ctx)

	hard := r.URL.Query().Get("hard") == "true"

	resp, err := h.photoSvc.Delete(ctx, photoID, hard, apiKeyID)
	if err != nil {
		h.handlePhotoError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// DownloadPhoto handles GET /api/v1/photos/{photo_id}/download
func (h *PhotoHandler) DownloadPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	photoIDStr := r.PathValue("photo_id")
	if photoIDStr == "" {
		h.writeError(w, http.StatusBadRequest, "photo_id is required", "BAD_REQUEST")
		return
	}

	photoID, err := vo.ParsePhotoID(photoIDStr)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid photo_id format", "INVALID_PHOTO_ID")
		return
	}

	photo, err := h.photoSvc.GetByID(ctx, photoID)
	if err != nil {
		h.handlePhotoError(w, err)
		return
	}

	signedURL, err := service.GenerateDownloadURL(h.gcsClient, photo.GCSObjectName(), downloadURLExpiryMinutes)
	if err != nil {
		h.handlePhotoError(w, err)
		return
	}

	http.Redirect(w, r, signedURL, http.StatusFound)
}

func (h *PhotoHandler) handlePhotoError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrPhotoNotFound) {
		h.writeError(w, http.StatusNotFound, "photo not found", "NOT_FOUND")
		return
	}
	if errors.Is(err, service.ErrPhotoDeleted) {
		h.writeError(w, http.StatusNotFound, "photo has been deleted", "NOT_FOUND")
		return
	}
	if errors.Is(err, service.ErrFileNotFound) {
		h.writeError(w, http.StatusNotFound, "photo file not found in storage", "FILE_NOT_FOUND")
		return
	}
	if ve, ok := err.(*model.ValidationError); ok {
		h.writeErrorWithDetails(w, http.StatusBadRequest, ve.Message, "VALIDATION_ERROR", ve.Field)
		return
	}
	if se, ok := err.(*service.ServiceError); ok {
		h.handleServiceError(w, se)
		return
	}
	h.handleServiceError(w, err)
}
