//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bina-marga/survey-photo/internal/model/dto/rest"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// doRequestWithAPIKeyString makes an HTTP request using a string API key
func doRequestWithAPIKeyString(t *testing.T, server *httptest.Server, method, path string, body io.Reader, apiKey string) *http.Response {
	t.Helper()

	url := server.URL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	return resp
}

// TestGetSignedUploadURL tests the Phase 1 upload endpoint
func TestGetSignedUploadURL_ValidRequest_Returns201(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup is handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build valid request
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100, // 100KB
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	// Make request
	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Parse response
	var respData rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &respData)

	assert.NotEmpty(t, respData.UploadToken)
	assert.NotEmpty(t, respData.SignedURL)
	assert.NotEmpty(t, respData.PhotoID)
	assert.True(t, respData.ExpiresAt.After(time.Now()))

	// Cleanup: delete the created photo
	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, respData.PhotoID)
	})
}

func TestGetSignedUploadURL_MissingAPIKey_Returns401(t *testing.T) {
	server := setupTestServer(t)

	// Build valid request
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	// Make request without API key
	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetSignedUploadURL_InvalidAPIKey_Returns401(t *testing.T) {
	server := setupTestServer(t)

	// Build valid request
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	// Make request with invalid API key
	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), "invalid-api-key")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetSignedUploadURL_ReadScopeOnly_Returns403(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with only read scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read"})

	// Build valid request
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	// Make request
	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestGetSignedUploadURL_MalformedJSON_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Make request with malformed JSON
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", strings.NewReader("{invalid json}"), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetSignedUploadURL_ReadScopeOnly_Returns403(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with only read scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read"})

	// Build valid request
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	// Make request
	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestGetSignedUploadURL_MalformedJSON_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Make request with malformed JSON
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", strings.NewReader("{invalid json}"), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSignedUploadURL_MissingFilename_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with empty filename
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "", // Empty filename
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Check error details
	respBody, _ := io.ReadAll(resp.Body)
	var errResp map[string]interface{}
	json.Unmarshal(respBody, &errResp)
	assert.Contains(t, errResp, "details")
}

func TestGetSignedUploadURL_MissingContentType_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with empty content type
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "", // Empty content type
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSignedUploadURL_FileTooLarge_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with file > 10MB
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 11 * 1024 * 1024, // 11MB - exceeds 10MB limit
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestGetSignedUploadURL_UnsupportedFormat_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with unsupported format (GIF)
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test.gif",
			ContentType:   "image/gif",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSignedUploadURL_InvalidLaneCode_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with invalid lane code
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "X99", // Invalid lane code
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSignedUploadURL_InvalidLatitude_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with invalid latitude (200 is out of range)
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  200, // Invalid: latitude must be between -90 and 90
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSignedUploadURL_InvalidLongitude_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with invalid longitude (300 is out of range)
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 300, // Invalid: longitude must be between -180 and 180
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSignedUploadURL_NegativeSTAValue_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with negative STA value
	negativeSTA := -1.0
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
			STAValue:  &negativeSTA, // Negative STA value
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetSignedUploadURL_QuotaExceeded_Returns429(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Create 10 pending uploads to exceed quota
	for i := 0; i < 10; i++ {
		reqBody := rest.GetSignedUploadURLRequest{
			FileMetadata: rest.FileMetadata{
				Filename:      fmt.Sprintf("test-photo-%d.jpg", i),
				ContentType:   "image/jpeg",
				FileSizeBytes: 1024 * 100,
			},
			PhotoAttributes: rest.PhotoAttributes{
				RouteID:   "NR-001",
				LaneCode:  "L1",
				Latitude:  -6.2088,
				Longitude: 106.8456,
			},
		}

		body, _ := json.Marshal(reqBody)
		resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to create pending upload %d: status %d", i, resp.StatusCode)
		}

		// Parse to get photo ID for cleanup
		var respData rest.GetSignedUploadURLResponse
		parseJSONResponse(t, resp, &respData)
		t.Cleanup(func() {
			ctx := context.Background()
			_ = server.photoRepo.HardDelete(ctx, respData.PhotoID)
		})
	}

	// Now try to create one more - should fail with 429
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo-11.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

// TestConfirmUpload tests the Phase 2 upload confirmation endpoint
func TestConfirmUpload_ValidToken_Returns200(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Phase 1: Get signed upload URL
	desc := "test photo"
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:     "NR-001",
			LaneCode:    "L1",
			Latitude:    -6.2088,
			Longitude:   106.8456,
			STAValue:    floatPtr(100.5),
			Description: &desc,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)

	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, uploadResp.PhotoID)
		cleanupGCSObject(t, server.gcsClient, getGCSObjectNameFromURL(uploadResp.SignedURL))
	})

	// Phase 2: Upload file to GCS
	testContent := []byte("test image content")
	uploadReq, _ := http.NewRequest(http.MethodPut, uploadResp.SignedURL, bytes.NewReader(testContent))
	uploadReq.Header.Set("Content-Type", "image/jpeg")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	uploadResp2, err := httpClient.Do(uploadReq)
	require.NoError(t, err)
	uploadResp2.Body.Close()
	require.Equal(t, http.StatusOK, uploadResp2.StatusCode)

	// Phase 3: Confirm upload
	confirmReq := rest.ConfirmUploadRequest{
		UploadToken: uploadResp.UploadToken,
	}

	confirmBody, _ := json.Marshal(confirmReq)
	confirmHttpResp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(confirmBody), apiKey.RawKey)
	defer confirmHttpResp.Body.Close()

	assert.Equal(t, http.StatusOK, confirmHttpResp.StatusCode)

	var confirmRespData rest.ConfirmUploadResponse
	parseJSONResponse(t, confirmHttpResp, &confirmRespData)
	assert.Equal(t, uploadResp.PhotoID, confirmRespData.PhotoID)
	assert.Equal(t, "Upload confirmed successfully", confirmRespData.Message)
}

func TestConfirmUpload_MalformedJSON_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Make request with malformed JSON
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", strings.NewReader("{invalid json}"), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestConfirmUpload_InvalidTokenFormat_Returns400(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with invalid token format
	reqBody := rest.ConfirmUploadRequest{
		UploadToken: vo.UploadToken("not-a-uuid"),
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestConfirmUpload_TokenNotFound_Returns404(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Build request with non-existent token
	nonExistentToken := vo.NewUploadToken()
	reqBody := rest.ConfirmUploadRequest{
		UploadToken: nonExistentToken,
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(body), apiKey.RawKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestConfirmUpload_TokenAlreadyUsed_Returns409(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Phase 1: Get signed upload URL
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)

	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, uploadResp.PhotoID)
		cleanupGCSObject(t, server.gcsClient, getGCSObjectNameFromURL(uploadResp.SignedURL))
	})

	// Phase 2: Upload file to GCS
	testContent := []byte("test image content")
	uploadReq, _ := http.NewRequest(http.MethodPut, uploadResp.SignedURL, bytes.NewReader(testContent))
	uploadReq.Header.Set("Content-Type", "image/jpeg")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	uploadResp2, err := httpClient.Do(uploadReq)
	require.NoError(t, err)
	uploadResp2.Body.Close()
	require.Equal(t, http.StatusOK, uploadResp2.StatusCode)

	// Phase 3: Confirm upload first time
	confirmReq := rest.ConfirmUploadRequest{
		UploadToken: uploadResp.UploadToken,
	}

	confirmBody, _ := json.Marshal(confirmReq)
	confirmHttpResp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(confirmBody), apiKey.RawKey)
	require.Equal(t, http.StatusOK, confirmHttpResp.StatusCode)
	confirmHttpResp.Body.Close()

	// Phase 4: Try to confirm again - should fail with 409
	confirmHttpResp2 := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(confirmBody), apiKey.RawKey)
	defer confirmHttpResp2.Body.Close()

	assert.Equal(t, http.StatusConflict, confirmHttpResp2.StatusCode)
}

func TestConfirmUpload_FileNotInGCS_Returns404(t *testing.T) {
	server := setupTestServer(t)

	// Create API key with write scope (cleanup handled internally)
	apiKey := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Phase 1: Get signed upload URL
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey.RawKey)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)

	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, uploadResp.PhotoID)
	})

	// Phase 2: Try to confirm WITHOUT uploading to GCS
	confirmReq := rest.ConfirmUploadRequest{
		UploadToken: uploadResp.UploadToken,
	}

	confirmBody, _ := json.Marshal(confirmReq)
	confirmHttpResp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(confirmBody), apiKey.RawKey)
	defer confirmHttpResp.Body.Close()

	// Should fail because file was never uploaded to GCS
	assert.Equal(t, http.StatusNotFound, confirmHttpResp.StatusCode)
}

func TestConfirmUpload_WrongAPIKey_Returns404(t *testing.T) {
	server := setupTestServer(t)

	// Create two API keys (cleanup handled internally for each)
	apiKey1 := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})
	apiKey2 := createTestAPIKey(t, server.apiKeyRepo, []string{"read", "write"})

	// Phase 1: Get signed upload URL with API key 1
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      "test-photo.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024 * 100,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey1)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)

	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, uploadResp.PhotoID)
		cleanupGCSObject(t, server.gcsClient, getGCSObjectNameFromURL(uploadResp.SignedURL))
	})

	// Phase 2: Upload file to GCS
	testContent := []byte("test image content")
	uploadReq, _ := http.NewRequest(http.MethodPut, uploadResp.SignedURL, bytes.NewReader(testContent))
	uploadReq.Header.Set("Content-Type", "image/jpeg")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	uploadResp2, err := httpClient.Do(uploadReq)
	require.NoError(t, err)
	uploadResp2.Body.Close()
	require.Equal(t, http.StatusOK, uploadResp2.StatusCode)

	// Phase 3: Try to confirm with API key 2 - should fail because wrong API key
	confirmReq := rest.ConfirmUploadRequest{
		UploadToken: uploadResp.UploadToken,
	}

	confirmBody, _ := json.Marshal(confirmReq)
	confirmHttpResp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(confirmBody), apiKey2)
	defer confirmHttpResp.Body.Close()

	// Should fail because token belongs to different API key
	assert.Equal(t, http.StatusNotFound, confirmHttpResp.StatusCode)
}

// Helper function to extract GCS object name from signed URL
func getGCSObjectNameFromURL(signedURL string) string {
	// Signed URL format: https://storage.googleapis.com/{bucket}/{object}?...
	// We need to extract the object name
	parts := strings.Split(signedURL, "/")
	if len(parts) < 5 {
		return ""
	}
	// Get the object name (everything after bucket name)
	objectParts := parts[4:]
	// Remove query string if present
	objectName := strings.Join(objectParts, "/")
	if idx := strings.Index(objectName, "?"); idx != -1 {
		objectName = objectName[:idx]
	}
	return objectName
}
