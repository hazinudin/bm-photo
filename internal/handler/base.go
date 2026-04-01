package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/bina-marga/survey-photo/internal/model"
)

type BaseHandler struct {
	logger *slog.Logger
}

func NewBaseHandler(logger *slog.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

func (h *BaseHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", "error", err)
	}
}

func (h *BaseHandler) writeError(w http.ResponseWriter, status int, errMsg string, code string) {
	h.writeJSON(w, status, ErrorResponse{
		Error: errMsg,
		Code:  code,
	})
}

func (h *BaseHandler) writeErrorWithDetails(w http.ResponseWriter, status int, errMsg string, code string, details string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   errMsg,
		Code:    code,
		Details: details,
	})
}

func (h *BaseHandler) handleValidationError(w http.ResponseWriter, err error) {
	if ve, ok := err.(*model.ValidationError); ok {
		h.writeErrorWithDetails(w, http.StatusBadRequest, ve.Message, "VALIDATION_ERROR", ve.Field)
		return
	}
	h.writeError(w, http.StatusBadRequest, err.Error(), "BAD_REQUEST")
}

func (h *BaseHandler) handleServiceError(w http.ResponseWriter, err error) {
	h.logger.Error("service error", "error", err)
	h.writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
}

func (h *BaseHandler) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				h.logger.Error("panic recovered", "error", err, "stack", string(debug.Stack()))
				h.writeError(w, http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
