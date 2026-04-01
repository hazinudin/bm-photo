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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
)

// TestGetPhoto_ValidID_Returns200 tests getting an existing completed photo
func TestGetPhoto_ValidID_Returns200(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createCompletedTestPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	resp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s", photoID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, readResponseBody(t, resp))

	var photoResp struct {
		PhotoID  string `json:"photo_id"`
		RouteID  string `json:"route_id"`
		LaneCode string `json:"lane_code"`
	}
	parseJSONResponse(t, resp, &photoResp)
	assert.Equal(t, string(photoID), photoResp.PhotoID)
	assert.Equal(t, "NR-001", photoResp.RouteID)
	assert.Equal(t, "L1", photoResp.LaneCode)
}

// TestGetPhoto_NotFound_Returns404 tests getting a non-existent photo
func TestGetPhoto_NotFound_Returns404(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	nonExistentID := vo.NewPhotoID()
	resp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s", nonExistentID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestGetPhoto_DeletedPhoto_Returns404 tests getting a soft-deleted photo
func TestGetPhoto_DeletedPhoto_Returns404(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createSoftDeletedPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	resp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s", photoID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestGetPhoto_InvalidID_Returns400 tests getting a photo with malformed UUID
func TestGetPhoto_InvalidID_Returns400(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})

	resp := doRequest(t, ts.Server, http.MethodGet, "/api/v1/photos/not-a-valid-uuid", nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestBrowsePhotos_ByRoute_Returns200 tests browsing photos by route ID
func TestBrowsePhotos_ByRoute_Returns200(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	// Create multiple photos for the route
	routeID := "NR-001"
	for i := 0; i < 3; i++ {
		photoID := createCompletedTestPhoto(t, ts, apiKey, routeID, "L1")
		defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, routeID, "L1"))
	}

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{"route_id": routeID}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, readResponseBody(t, resp))

	var browseResp struct {
		Photos []struct {
			PhotoID string `json:"photo_id"`
			RouteID string `json:"route_id"`
		} `json:"photos"`
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	parseJSONResponse(t, resp, &browseResp)
	assert.GreaterOrEqual(t, len(browseResp.Photos), 3, "expected at least 3 photos")
	assert.Equal(t, 3, int(browseResp.Pagination.TotalCount))
}

// TestBrowsePhotos_ByRouteAndSTA_Returns200 tests browsing photos by route and STA range
func TestBrowsePhotos_ByRouteAndSTA_Returns200(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	// Create photos with different STA values
	routeID := "NR-002"
	createPhotoWithSTA(t, ts, apiKey, routeID, "L1", 5.0)
	createPhotoWithSTA(t, ts, apiKey, routeID, "L1", 10.0)
	createPhotoWithSTA(t, ts, apiKey, routeID, "L1", 15.0)

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{
		"route_id":  routeID,
		"sta_start": "7.0",
		"sta_end":   "12.0",
	}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, readResponseBody(t, resp))

	var browseResp struct {
		Photos []struct {
			PhotoID  string  `json:"photo_id"`
			RouteID  string  `json:"route_id"`
			STAValue float64 `json:"sta_value"`
		} `json:"photos"`
	}
	parseJSONResponse(t, resp, &browseResp)
	assert.Equal(t, 1, len(browseResp.Photos), "expected 1 photo in STA range 7-12")
	assert.Equal(t, 10.0, browseResp.Photos[0].STAValue)
}

// TestBrowsePhotos_ByRouteAndLane_Returns200 tests browsing photos by route and lane
func TestBrowsePhotos_ByRouteAndLane_Returns200(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	routeID := "NR-003"
	createCompletedTestPhoto(t, ts, apiKey, routeID, "L1")
	createCompletedTestPhoto(t, ts, apiKey, routeID, "L2")
	createCompletedTestPhoto(t, ts, apiKey, routeID, "R1")

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{
		"route_id":  routeID,
		"lane_code": "L1",
	}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, readResponseBody(t, resp))

	var browseResp struct {
		Photos []struct {
			PhotoID  string `json:"photo_id"`
			LaneCode string `json:"lane_code"`
		} `json:"photos"`
	}
	parseJSONResponse(t, resp, &browseResp)
	for _, photo := range browseResp.Photos {
		assert.Equal(t, "L1", photo.LaneCode, "expected all photos to be L1 lane")
	}
}

// TestBrowsePhotos_MissingRouteID_Returns400 tests browsing without route_id
func TestBrowsePhotos_MissingRouteID_Returns400(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})

	resp := doRequest(t, ts.Server, http.MethodGet, "/api/v1/photos", nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestBrowsePhotos_InvalidSTARange_Returns400 tests browsing with sta_start > sta_end
func TestBrowsePhotos_InvalidSTARange_Returns400(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{
		"route_id":  "NR-001",
		"sta_start": "20.0",
		"sta_end":   "10.0",
	}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestBrowsePhotos_InvalidLaneCode_Returns400 tests browsing with invalid lane_code
func TestBrowsePhotos_InvalidLaneCode_Returns400(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{
		"route_id":  "NR-001",
		"lane_code": "X99",
	}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestBrowsePhotos_Pagination_Defaults tests pagination defaults
func TestBrowsePhotos_Pagination_Defaults(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	// Create more than default per_page photos
	routeID := "NR-PAGINATE"
	for i := 0; i < 25; i++ {
		photoID := createCompletedTestPhoto(t, ts, apiKey, routeID, "L1")
		defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, routeID, "L1"))
	}

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{"route_id": routeID}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var browseResp struct {
		Photos []struct {
			PhotoID string `json:"photo_id"`
		} `json:"photos"`
		Pagination struct {
			CurrentPage int `json:"current_page"`
			PerPage     int `json:"per_page"`
			TotalCount  int `json:"total_count"`
		} `json:"pagination"`
	}
	parseJSONResponse(t, resp, &browseResp)

	// Default per_page is 20
	assert.Equal(t, 20, browseResp.Pagination.PerPage, "expected default per_page of 20")
	assert.Equal(t, 1, browseResp.Pagination.CurrentPage, "expected current_page of 1")
	assert.Equal(t, 25, browseResp.Pagination.TotalCount, "expected total_count of 25")
}

// TestBrowsePhotos_Pagination_MaxPerPage tests per_page capping at 100
func TestBrowsePhotos_Pagination_MaxPerPage(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	routeID := "NR-MAXPAGE"
	for i := 0; i < 5; i++ {
		photoID := createCompletedTestPhoto(t, ts, apiKey, routeID, "L1")
		defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, routeID, "L1"))
	}

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{
		"route_id": routeID,
		"per_page": "200", // Request more than max
	}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var browseResp struct {
		Pagination struct {
			PerPage int `json:"per_page"`
		} `json:"pagination"`
	}
	parseJSONResponse(t, resp, &browseResp)

	// Max per_page is 100
	assert.Equal(t, 100, browseResp.Pagination.PerPage, "expected per_page capped at 100")
}

// TestBrowsePhotos_NoResults_ReturnsEmptyList tests browsing with no results
func TestBrowsePhotos_NoResults_ReturnsEmptyList(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})

	resp := doRequestWithQuery(t, ts.Server, http.MethodGet, "/api/v1/photos", map[string]string{"route_id": "NONEXISTENT"}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var browseResp struct {
		Photos     []interface{} `json:"photos"`
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	parseJSONResponse(t, resp, &browseResp)
	assert.Empty(t, browseResp.Photos)
	assert.Equal(t, 0, browseResp.Pagination.TotalCount)
}

// TestUpdatePhoto_ValidUpdate_Returns200 tests updating photo metadata
func TestUpdatePhoto_ValidUpdate_Returns200(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createCompletedTestPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	newDescription := "Updated description"
	newTags := []string{"updated", "tags"}

	updateReq := map[string]interface{}{
		"description": newDescription,
		"tags":        newTags,
	}
	body, _ := json.Marshal(updateReq)

	resp := doRequestWithBody(t, ts.Server, http.MethodPatch, fmt.Sprintf("/api/v1/photos/%s", photoID), bytes.NewReader(body), apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, readResponseBody(t, resp))

	var updateResp struct {
		PhotoID     string   `json:"photo_id"`
		Description *string  `json:"description"`
		Tags        []string `json:"tags"`
	}
	parseJSONResponse(t, resp, &updateResp)
	assert.Equal(t, string(photoID), updateResp.PhotoID)
	assert.Equal(t, newDescription, *updateResp.Description)
	assert.Equal(t, newTags, updateResp.Tags)
}

// TestUpdatePhoto_InvalidLaneCode_Returns400 tests updating with invalid lane_code
func TestUpdatePhoto_InvalidLaneCode_Returns400(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createCompletedTestPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	updateReq := map[string]interface{}{
		"lane_code": "X99",
	}
	body, _ := json.Marshal(updateReq)

	resp := doRequestWithBody(t, ts.Server, http.MethodPatch, fmt.Sprintf("/api/v1/photos/%s", photoID), bytes.NewReader(body), apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestUpdatePhoto_PhotoNotFound_Returns404 tests updating a non-existent photo
func TestUpdatePhoto_PhotoNotFound_Returns404(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	updateReq := map[string]interface{}{
		"description": "New description",
	}
	body, _ := json.Marshal(updateReq)

	nonExistentID := vo.NewPhotoID()
	resp := doRequestWithBody(t, ts.Server, http.MethodPatch, fmt.Sprintf("/api/v1/photos/%s", nonExistentID), bytes.NewReader(body), apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestUpdatePhoto_ReadScopeOnly_Returns403 tests updating with read-only API key
func TestUpdatePhoto_ReadScopeOnly_Returns403(t *testing.T) {
	ts := setupTestServer(t)
	readOnlyKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})
	fullKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	photoID := createCompletedTestPhoto(t, ts, fullKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	updateReq := map[string]interface{}{
		"description": "New description",
	}
	body, _ := json.Marshal(updateReq)

	// Try to update with read-only key
	resp := doRequestWithBody(t, ts.Server, http.MethodPatch, fmt.Sprintf("/api/v1/photos/%s", photoID), bytes.NewReader(body), readOnlyKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestDeletePhoto_SoftDelete_Returns200 tests soft deleting a photo
func TestDeletePhoto_SoftDelete_Returns200(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createCompletedTestPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	resp := doRequest(t, ts.Server, http.MethodDelete, fmt.Sprintf("/api/v1/photos/%s", photoID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, readResponseBody(t, resp))

	var deleteResp struct {
		PhotoID      string `json:"photo_id"`
		DeletionType string `json:"deletion_type"`
	}
	parseJSONResponse(t, resp, &deleteResp)
	assert.Equal(t, string(photoID), deleteResp.PhotoID)
	assert.Equal(t, "soft", deleteResp.DeletionType)

	// Verify photo is no longer accessible via GET
	getResp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s", photoID), nil, apiKey)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

// TestDeletePhoto_HardDelete_Returns200 tests hard deleting a photo
func TestDeletePhoto_HardDelete_Returns200(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createCompletedTestPhoto(t, ts, apiKey, "NR-001", "L1")
	gcsObjectName := getGCSObjectNameForPhoto(photoID, "NR-001", "L1")

	// Delete with hard=true
	resp := doRequestWithQuery(t, ts.Server, http.MethodDelete, fmt.Sprintf("/api/v1/photos/%s", photoID), map[string]string{"hard": "true"}, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK, got %d: %s", resp.StatusCode, readResponseBody(t, resp))

	var deleteResp struct {
		PhotoID      string `json:"photo_id"`
		DeletionType string `json:"deletion_type"`
	}
	parseJSONResponse(t, resp, &deleteResp)
	assert.Equal(t, string(photoID), deleteResp.PhotoID)
	assert.Equal(t, "hard", deleteResp.DeletionType)

	// Verify photo is completely gone
	getResp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s", photoID), nil, apiKey)
	defer getResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)

	// Cleanup GCS object if still exists
	defer cleanupGCSObject(t, ts.gcsClient, gcsObjectName)
}

// TestDeletePhoto_AlreadySoftDeleted_Returns404 tests deleting an already soft-deleted photo
func TestDeletePhoto_AlreadySoftDeleted_Returns404(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createSoftDeletedPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	resp := doRequest(t, ts.Server, http.MethodDelete, fmt.Sprintf("/api/v1/photos/%s", photoID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDeletePhoto_ReadScopeOnly_Returns403 tests deleting with read-only API key
func TestDeletePhoto_ReadScopeOnly_Returns403(t *testing.T) {
	ts := setupTestServer(t)
	readOnlyKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})
	fullKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})

	photoID := createCompletedTestPhoto(t, ts, fullKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	resp := doRequest(t, ts.Server, http.MethodDelete, fmt.Sprintf("/api/v1/photos/%s", photoID), nil, readOnlyKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestDownloadPhoto_ValidID_Returns302 tests downloading an existing photo
func TestDownloadPhoto_ValidID_Returns302(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createCompletedTestPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	resp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s/download", photoID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusFound, resp.StatusCode, "expected 302 redirect, got %d", resp.StatusCode)
	// Location header should contain GCS signed URL
	location := resp.Header.Get("Location")
	assert.Contains(t, location, "storage.googleapis.com", "expected redirect to GCS signed URL")
}

// TestDownloadPhoto_NotFound_Returns404 tests downloading a non-existent photo
func TestDownloadPhoto_NotFound_Returns404(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read"})

	nonExistentID := vo.NewPhotoID()
	resp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s/download", nonExistentID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDownloadPhoto_DeletedPhoto_Returns404 tests downloading a soft-deleted photo
func TestDownloadPhoto_DeletedPhoto_Returns404(t *testing.T) {
	ts := setupTestServer(t)
	apiKey := createTestAPIKey(t, ts.apiKeyRepo, []string{"read", "write", "admin"})
	photoID := createSoftDeletedPhoto(t, ts, apiKey, "NR-001", "L1")
	defer cleanupGCSObject(t, ts.gcsClient, getGCSObjectNameForPhoto(photoID, "NR-001", "L1"))

	resp := doRequest(t, ts.Server, http.MethodGet, fmt.Sprintf("/api/v1/photos/%s/download", photoID), nil, apiKey)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// Helper function to create a completed test photo
func createCompletedTestPhoto(t *testing.T, ts *testServer, apiKey, routeID, laneCode string) vo.PhotoID {
	t.Helper()

	ctx := context.Background()

	// Generate unique identifiers
	photoID := vo.NewPhotoID()
	gcsObjectName := fmt.Sprintf("photos/2026/%s/%s_2026_%s_%s.jpg",
		routeID, routeID, laneCode, photoID.String()[:8])

	// Create photo params
	params := entity.PhotoParams{
		RouteID:          routeID,
		LaneCode:         laneCode,
		GCSObjectName:    gcsObjectName,
		FileFormat:       vo.FileFormatJPEG,
		FileSizeBytes:    1024 * 100,
		OriginalFilename: strPtr("test_photo.jpg"),
		UploadToken:      vo.NewUploadToken(),
		UploadedBy:       "test-api-key",
	}

	photo, err := entity.NewPhoto(params)
	require.NoError(t, err, "failed to create photo entity")

	// Set coordinates
	err = photo.SetCoordinates(-6.2088, 106.8456)
	require.NoError(t, err, "failed to set coordinates")

	// Set STA value
	err = photo.SetSTA(5.5, vo.STASourceUserProvided)
	require.NoError(t, err, "failed to set STA")

	// Mark as completed
	err = photo.MarkUploadCompleted()
	require.NoError(t, err, "failed to mark upload completed")

	// Create in database
	err = ts.photoRepo.Create(ctx, photo)
	require.NoError(t, err, "failed to create photo in database")

	// Upload a test file to GCS
	testContent := []byte("test image content for photo")
	signedURL, err := ts.gcsClient.GenerateSignedURL(gcsObjectName, "image/jpeg", 60)
	if err == nil {
		req, err := http.NewRequest(http.MethodPut, signedURL, bytes.NewReader(testContent))
		if err == nil {
			req.Header.Set("Content-Type", "image/jpeg")
			client := &http.Client{Timeout: 30 * 1000000000} // 30 seconds
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}
	}

	return photoID
}

// Helper function to create a soft-deleted photo
func createSoftDeletedPhoto(t *testing.T, ts *testServer, apiKey, routeID, laneCode string) vo.PhotoID {
	t.Helper()

	ctx := context.Background()

	// Create a completed photo first
	photoID := createCompletedTestPhoto(t, ts, apiKey, routeID, laneCode)

	// Soft delete it
	err := ts.photoRepo.SoftDelete(ctx, photoID, "test-api-key")
	require.NoError(t, err, "failed to soft delete photo")

	return photoID
}

// Helper function to create a photo with specific STA value
func createPhotoWithSTA(t *testing.T, ts *testServer, apiKey, routeID, laneCode string, staValue float64) vo.PhotoID {
	t.Helper()

	ctx := context.Background()

	photoID := vo.NewPhotoID()
	gcsObjectName := fmt.Sprintf("photos/2026/%s/%s_2026_%s_%s.jpg",
		routeID, routeID, laneCode, photoID.String()[:8])

	params := entity.PhotoParams{
		RouteID:          routeID,
		LaneCode:         laneCode,
		GCSObjectName:    gcsObjectName,
		FileFormat:       vo.FileFormatJPEG,
		FileSizeBytes:    1024 * 100,
		OriginalFilename: strPtr("test_photo.jpg"),
		UploadToken:      vo.NewUploadToken(),
		UploadedBy:       "test-api-key",
	}

	photo, err := entity.NewPhoto(params)
	require.NoError(t, err, "failed to create photo entity")

	err = photo.SetCoordinates(-6.2088, 106.8456)
	require.NoError(t, err, "failed to set coordinates")

	err = photo.SetSTA(staValue, vo.STASourceUserProvided)
	require.NoError(t, err, "failed to set STA")

	err = photo.MarkUploadCompleted()
	require.NoError(t, err, "failed to mark upload completed")

	err = ts.photoRepo.Create(ctx, photo)
	require.NoError(t, err, "failed to create photo in database")

	// Upload test file to GCS
	testContent := []byte("test image content")
	signedURL, err := ts.gcsClient.GenerateSignedURL(gcsObjectName, "image/jpeg", 60)
	if err == nil {
		req, err := http.NewRequest(http.MethodPut, signedURL, bytes.NewReader(testContent))
		if err == nil {
			req.Header.Set("Content-Type", "image/jpeg")
			client := &http.Client{Timeout: 30 * 1000000000}
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}
	}

	return photoID
}

// Helper function to get GCS object name for a photo
func getGCSObjectNameForPhoto(photoID vo.PhotoID, routeID, laneCode string) string {
	return fmt.Sprintf("photos/2026/%s/%s_2026_%s_%s.jpg",
		routeID, routeID, laneCode, photoID.String()[:8])
}

// doRequestWithBody makes an HTTP request with a body
func doRequestWithBody(t *testing.T, server *httptest.Server, method, path string, body io.Reader, apiKey string) *http.Response {
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

	client := &http.Client{Timeout: 30 * 1000000000}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	return resp
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}
