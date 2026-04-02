//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

	// Read actual test photo
	testPhotoFilename := getTestPhotoFilename(0)
	testContent := readTestPhoto(t, testPhotoFilename)
	fileSize := int64(len(testContent))

	// Build valid request
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      testPhotoFilename,
			ContentType:   "image/jpeg",
			FileSizeBytes: fileSize,
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
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)
	resp.Body.Close()

	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, uploadResp.PhotoID)
		cleanupGCSObject(t, server.gcsClient, getGCSObjectNameFromURL(uploadResp.SignedURL))
	})

	// Phase 2: Upload actual file to GCS
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

	// Read actual test photo
	testPhotoFilename := getTestPhotoFilename(1)
	testContent := readTestPhoto(t, testPhotoFilename)
	fileSize := int64(len(testContent))

	// Phase 1: Get signed upload URL
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      testPhotoFilename,
			ContentType:   "image/jpeg",
			FileSizeBytes: fileSize,
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

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)
	resp.Body.Close()

	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, uploadResp.PhotoID)
		cleanupGCSObject(t, server.gcsClient, getGCSObjectNameFromURL(uploadResp.SignedURL))
	})

	// Phase 2: Upload actual file to GCS
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

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)
	resp.Body.Close()

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

	// Read actual test photo
	testPhotoFilename := getTestPhotoFilename(2)
	testContent := readTestPhoto(t, testPhotoFilename)
	fileSize := int64(len(testContent))

	// Phase 1: Get signed upload URL with API key 1
	reqBody := rest.GetSignedUploadURLRequest{
		FileMetadata: rest.FileMetadata{
			Filename:      testPhotoFilename,
			ContentType:   "image/jpeg",
			FileSizeBytes: fileSize,
		},
		PhotoAttributes: rest.PhotoAttributes{
			RouteID:   "NR-001",
			LaneCode:  "L1",
			Latitude:  -6.2088,
			Longitude: 106.8456,
		},
	}

	body, _ := json.Marshal(reqBody)
	resp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/upload-url", bytes.NewReader(body), apiKey1.RawKey)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var uploadResp rest.GetSignedUploadURLResponse
	parseJSONResponse(t, resp, &uploadResp)
	resp.Body.Close()

	t.Cleanup(func() {
		ctx := context.Background()
		_ = server.photoRepo.HardDelete(ctx, uploadResp.PhotoID)
		cleanupGCSObject(t, server.gcsClient, getGCSObjectNameFromURL(uploadResp.SignedURL))
	})

	// Phase 2: Upload actual file to GCS
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
	confirmHttpResp := doRequestWithAPIKeyString(t, server.Server, http.MethodPost, "/api/v1/photos/confirm", bytes.NewReader(confirmBody), apiKey2.RawKey)
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

// getTestPhotoPath returns the path to a test photo in the testphotos folder
func getTestPhotoPath(filename string) string {
	return filepath.Join("..", "..", "testphotos", filename)
}

// readTestPhoto reads the contents of a test photo file
func readTestPhoto(t *testing.T, filename string) []byte {
	t.Helper()

	path := getTestPhotoPath(filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read test photo %s: %v", path, err)
	}
	return data
}

// getTestPhotoFilename returns a test photo filename from the testphotos folder
func getTestPhotoFilename(index int) string {
	photos := []string{
		"M_0.jpg",
		"M_10.jpg",
		"M_100.jpg",
		"M_1000.jpg",
		"M_10000.jpg",
		"M_10010.jpg",
		"M_10020.jpg",
	}
	if index >= len(photos) {
		index = 0
	}
	return photos[index]
}

// floatPtr returns a pointer to a float64 value
func floatPtr(v float64) *float64 {
	return &v
}
