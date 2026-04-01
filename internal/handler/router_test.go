//go:build integration

package handler

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// mockUploadService implements service.UploadService for testing
type mockUploadService struct{}

func (m *mockUploadService) GetSignedURL(ctx context.Context, req *rest.GetSignedUploadURLRequest, apiKeyID string) (*rest.GetSignedUploadURLResponse, error) {
	return nil, nil
}

func (m *mockUploadService) ConfirmUpload(ctx context.Context, token vo.UploadToken, apiKeyID string) (*rest.ConfirmUploadResponse, error) {
	return nil, nil
}

// mockPhotoService implements service.PhotoService for testing
type mockPhotoService struct{}

func (m *mockPhotoService) GetByID(ctx context.Context, id vo.PhotoID) (*entity.Photo, error) {
	return nil, nil
}

func (m *mockPhotoService) Browse(ctx context.Context, filter repository.BrowseFilter) (*rest.BrowsePhotosResponse, error) {
	return nil, nil
}

func (m *mockPhotoService) Search(ctx context.Context, filter repository.SearchFilter) (*rest.SearchPhotosResponse, error) {
	return nil, nil
}

func (m *mockPhotoService) Update(ctx context.Context, id vo.PhotoID, req *rest.UpdatePhotoRequest) (*rest.UpdatePhotoResponse, error) {
	return nil, nil
}

func (m *mockPhotoService) Delete(ctx context.Context, id vo.PhotoID, hard bool, apiKeyID string) (*rest.DeletePhotoResponse, error) {
	return nil, nil
}

// mockGCSClient implements service.GCSClient for testing
type mockGCSClient struct{}

func (m *mockGCSClient) GenerateSignedURL(objectName, contentType string, expiryMinutes int) (string, error) {
	return "", nil
}

func (m *mockGCSClient) FileExists(objectName string) (bool, error) {
	return false, nil
}

func (m *mockGCSClient) DeleteFile(objectName string) error {
	return nil
}

// mockAuthServiceForRouter implements service.AuthService for router tests
type mockAuthServiceForRouter struct{}

func (m *mockAuthServiceForRouter) ValidateAPIKey(ctx context.Context, key string) (*repository.APIKey, error) {
	return nil, nil
}

func (m *mockAuthServiceForRouter) CheckScope(apiKey *repository.APIKey, scope string) error {
	return nil
}

// mockDBPingerForRouter implements a simple pinger for router tests
type mockDBPingerForRouter struct{}

func (m *mockDBPingerForRouter) Ping(ctx context.Context) error {
	return nil
}

func TestRouter_CORS_Preflight(t *testing.T) {
	logger := slog.Default()
	router := NewRouter(
		&mockUploadService{},
		&mockPhotoService{},
		&mockAuthServiceForRouter{},
		&mockGCSClient{},
		&mockDBPingerForRouter{},
		logger,
	)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/photos", nil)
	// Add CORS preflight headers
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-API-Key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Check CORS headers
	if allowOrigin := w.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", allowOrigin)
	}

	if allowMethods := w.Header().Get("Access-Control-Allow-Methods"); allowMethods == "" {
		t.Error("expected Access-Control-Allow-Methods header to be set")
	}

	if allowHeaders := w.Header().Get("Access-Control-Allow-Headers"); allowHeaders == "" {
		t.Error("expected Access-Control-Allow-Headers header to be set")
	}
}

func TestRouter_CORS_HeadersOnResponse(t *testing.T) {
	logger := slog.Default()
	router := NewRouter(
		&mockUploadService{},
		&mockPhotoService{},
		&mockAuthServiceForRouter{},
		&mockGCSClient{},
		&mockDBPingerForRouter{},
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Check CORS headers on normal response
	if allowOrigin := w.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", allowOrigin)
	}
}

func TestRouter_UnknownRoute_Returns404(t *testing.T) {
	logger := slog.Default()
	router := NewRouter(
		&mockUploadService{},
		&mockPhotoService{},
		&mockAuthServiceForRouter{},
		&mockGCSClient{},
		&mockDBPingerForRouter{},
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestRouter_MiddlewareChain_Order(t *testing.T) {
	logger := slog.Default()
	router := NewRouter(
		&mockUploadService{},
		&mockPhotoService{},
		&mockAuthServiceForRouter{},
		&mockGCSClient{},
		&mockDBPingerForRouter{},
		logger,
	)

	// Request without API key to a protected endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/photos", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 401 Unauthorized because API key is missing
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
