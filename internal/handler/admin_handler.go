package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/service"
)

type AdminHandler struct {
	*BaseHandler
	adminSvc service.AdminService
}

func NewAdminHandler(adminSvc service.AdminService, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		BaseHandler: NewBaseHandler(logger),
		adminSvc:    adminSvc,
	}
}

// CreateAPIKey handles POST /api/v1/admin/api-keys
func (h *AdminHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req rest.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", "BAD_REQUEST")
		return
	}

	if len(req.Scopes) == 0 {
		h.writeError(w, http.StatusBadRequest, "at least one scope is required", "INVALID_SCOPES")
		return
	}

	resp, err := h.adminSvc.CreateAPIKey(ctx, &req)
	if err != nil {
		h.handleAdminError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, resp)
}

// ListAPIKeys handles GET /api/v1/admin/api-keys
func (h *AdminHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	activeOnly := r.URL.Query().Get("active_only") == "true"

	resp, err := h.adminSvc.ListAPIKeys(ctx, activeOnly)
	if err != nil {
		h.handleAdminError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// RevokeAPIKey handles DELETE /api/v1/admin/api-keys/{key_id}
func (h *AdminHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	keyID := r.PathValue("key_id")

	if keyID == "" {
		h.writeError(w, http.StatusBadRequest, "key_id is required", "BAD_REQUEST")
		return
	}

	resp, err := h.adminSvc.RevokeAPIKey(ctx, keyID)
	if err != nil {
		h.handleAdminError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *AdminHandler) handleAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAPIKeyNotFound):
		h.writeError(w, http.StatusNotFound, "API key not found", "API_KEY_NOT_FOUND")
	case errors.Is(err, service.ErrAPIKeyCreationFailed):
		h.writeError(w, http.StatusInternalServerError, "failed to create API key", "CREATION_FAILED")
	case errors.Is(err, service.ErrInvalidScope):
		h.writeError(w, http.StatusBadRequest, err.Error(), "INVALID_SCOPE")
	default:
		h.writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
