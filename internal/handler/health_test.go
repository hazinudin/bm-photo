//go:build integration

package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockDBPinger implements a mock dbPinger for testing
type mockDBPinger struct {
	pingErr error
}

func (m *mockDBPinger) Ping(ctx context.Context) error {
	return m.pingErr
}

// mockAuthService implements service.AuthService for testing
type mockAuthService struct{}

func (m *mockAuthService) ValidateAPIKey(ctx context.Context, key string) (any, error) {
	return nil, nil
}

func (m *mockAuthService) CheckScope(apiKey any, scope string) error {
	return nil
}

func TestHealthCheck_Returns200(t *testing.T) {
	pinger := &mockDBPinger{pingErr: nil}
	handler := NewHealthHandler(pinger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp["status"])
	}
}

func TestHealthCheck_NoAuthRequired(t *testing.T) {
	pinger := &mockDBPinger{pingErr: nil}
	handler := NewHealthHandler(pinger)

	// Request without API key header
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.HealthCheck(w, req)

	// Should still return 200 - no auth required for health endpoint
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestReadinessCheck_DBUp_Returns200(t *testing.T) {
	pinger := &mockDBPinger{pingErr: nil}
	handler := NewHealthHandler(pinger)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.ReadinessCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ready" {
		t.Errorf("expected status 'ready', got '%s'", resp["status"])
	}
}

func TestReadinessCheck_DBDown_Returns503(t *testing.T) {
	pinger := &mockDBPinger{pingErr: errors.New("connection refused")}
	handler := NewHealthHandler(pinger)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.ReadinessCheck(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "not_ready" {
		t.Errorf("expected status 'not_ready', got '%s'", resp["status"])
	}
}
