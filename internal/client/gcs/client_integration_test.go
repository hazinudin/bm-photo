//go:build integration
// +build integration

package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getTestConfig returns config for integration tests
func getTestConfig(t *testing.T) Config {
	t.Helper()

	config, err := LoadConfigFromEnv()
	require.NoError(t, err, "Failed to load config from environment")

	return config
}

// getTestObjectName returns a unique test object name
func getTestObjectName(t *testing.T, suffix string) string {
	t.Helper()
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%sintegration-test-%d-%s",
		os.Getenv("GCS_TEST_PREFIX"),
		timestamp,
		suffix,
	)
}

// TestGenerateSignedURL tests signed URL generation
func TestGenerateSignedURL(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close()

	t.Run("generate upload URL", func(t *testing.T) {
		objectName := getTestObjectName(t, "upload.jpg")
		signedURL, err := client.GenerateSignedURL(objectName, "image/jpeg", 15)

		require.NoError(t, err)
		assert.NotEmpty(t, signedURL)
		assert.Contains(t, signedURL, "storage.googleapis.com")
		assert.Contains(t, signedURL, objectName)
		assert.Contains(t, signedURL, "X-Goog-Signature")
	})

	t.Run("generate download URL", func(t *testing.T) {
		objectName := getTestObjectName(t, "download.jpg")
		signedURL, err := client.GenerateSignedURL(objectName, "", 15)

		require.NoError(t, err)
		assert.NotEmpty(t, signedURL)
		assert.Contains(t, signedURL, "storage.googleapis.com")
	})

	t.Run("default expiry", func(t *testing.T) {
		objectName := getTestObjectName(t, "default.jpg")
		signedURL, err := client.GenerateSignedURL(objectName, "image/jpeg", 0)

		require.NoError(t, err)
		assert.NotEmpty(t, signedURL)
	})

	t.Run("empty object name fails", func(t *testing.T) {
		_, err := client.GenerateSignedURL("", "image/jpeg", 15)
		assert.Error(t, err)
	})
}

// TestUploadViaSignedURL tests end-to-end upload workflow
func TestUploadViaSignedURL(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close()

	objectName := getTestObjectName(t, "upload-test.jpg")
	testContent := []byte("test image content for upload")

	t.Run("upload file via signed URL", func(t *testing.T) {
		// Generate signed upload URL
		signedURL, err := client.GenerateSignedURL(objectName, "image/jpeg", 15)
		require.NoError(t, err)

		// Upload via HTTP PUT
		req, err := http.NewRequest(http.MethodPut, signedURL, bytes.NewReader(testContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "image/jpeg")

		httpClient := &http.Client{Timeout: 30 * time.Second}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Cleanup
		defer client.DeleteFile(objectName)
	})

	t.Run("verify uploaded file exists", func(t *testing.T) {
		// First upload a file
		signedURL, err := client.GenerateSignedURL(objectName, "image/jpeg", 15)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, signedURL, bytes.NewReader(testContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "image/jpeg")

		httpClient := &http.Client{Timeout: 30 * time.Second}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify file exists
		exists, err := client.FileExists(objectName)
		require.NoError(t, err)
		assert.True(t, exists)

		// Cleanup
		client.DeleteFile(objectName)
	})
}

// TestFileExists tests file existence checking
func TestFileExists(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close()

	t.Run("file does not exist", func(t *testing.T) {
		objectName := getTestObjectName(t, "nonexistent.txt")
		exists, err := client.FileExists(objectName)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("empty object name fails", func(t *testing.T) {
		_, err := client.FileExists("")
		assert.Error(t, err)
	})

	t.Run("file exists after upload", func(t *testing.T) {
		objectName := getTestObjectName(t, "exists-test.txt")
		testContent := []byte("test content")

		// Upload file
		signedURL, err := client.GenerateSignedURL(objectName, "text/plain", 15)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, signedURL, bytes.NewReader(testContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")

		httpClient := &http.Client{Timeout: 30 * time.Second}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		// Check existence
		exists, err := client.FileExists(objectName)
		require.NoError(t, err)
		assert.True(t, exists)

		// Cleanup
		client.DeleteFile(objectName)
	})
}

// TestDeleteFile tests file deletion
func TestDeleteFile(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close()

	t.Run("delete existing file", func(t *testing.T) {
		objectName := getTestObjectName(t, "delete-test.txt")
		testContent := []byte("content to delete")

		// Upload file first
		signedURL, err := client.GenerateSignedURL(objectName, "text/plain", 15)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, signedURL, bytes.NewReader(testContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "text/plain")

		httpClient := &http.Client{Timeout: 30 * time.Second}
		resp, err := httpClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		// Verify file exists
		exists, err := client.FileExists(objectName)
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete file
		err = client.DeleteFile(objectName)
		require.NoError(t, err)

		// Verify file no longer exists
		exists, err = client.FileExists(objectName)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent file is idempotent", func(t *testing.T) {
		objectName := getTestObjectName(t, "already-deleted.txt")

		err := client.DeleteFile(objectName)
		assert.NoError(t, err)
	})

	t.Run("empty object name fails", func(t *testing.T) {
		err := client.DeleteFile("")
		assert.Error(t, err)
	})
}

// TestEndToEndUploadConfirm tests complete upload workflow
func TestEndToEndUploadConfirm(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	client, err := NewClient(ctx, config)
	require.NoError(t, err)
	defer client.Close()

	objectName := getTestObjectName(t, "e2e-test.jpg")
	testContent := []byte("end to end test content")

	// Phase 1: Generate signed URL
	signedURL, err := client.GenerateSignedURL(objectName, "image/jpeg", 15)
	require.NoError(t, err, "Phase 1: Generate signed URL failed")
	assert.NotEmpty(t, signedURL)

	// Phase 2: Upload file to GCS
	req, err := http.NewRequest(http.MethodPut, signedURL, bytes.NewReader(testContent))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "image/jpeg")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Phase 2: Upload failed: %s", string(body))

	// Phase 3: Verify file exists
	exists, err := client.FileExists(objectName)
	require.NoError(t, err, "Phase 3: File existence check failed")
	assert.True(t, exists, "Phase 3: File should exist after upload")

	// Phase 4: Generate download URL and verify access
	downloadURL, err := client.GenerateSignedURL(objectName, "", 15)
	require.NoError(t, err, "Phase 4: Generate download URL failed")

	// Download and verify content
	downloadResp, err := httpClient.Get(downloadURL)
	require.NoError(t, err)
	defer downloadResp.Body.Close()

	downloadedContent, err := io.ReadAll(downloadResp.Body)
	require.NoError(t, err)
	assert.Equal(t, testContent, downloadedContent, "Phase 4: Downloaded content mismatch")

	// Phase 5: Delete file
	err = client.DeleteFile(objectName)
	require.NoError(t, err, "Phase 5: Delete failed")

	// Phase 6: Verify deletion
	exists, err = client.FileExists(objectName)
	require.NoError(t, err, "Phase 6: File existence check failed")
	assert.False(t, exists, "Phase 6: File should not exist after deletion")
}

// TestClientClose tests client cleanup
func TestClientClose(t *testing.T) {
	ctx := context.Background()
	config := getTestConfig(t)

	client, err := NewClient(ctx, config)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	ctx := context.Background()

	t.Run("valid credentials", func(t *testing.T) {
		config := getTestConfig(t)
		client, err := NewClient(ctx, config)
		require.NoError(t, err)
		client.Close()
	})

	t.Run("invalid credentials path", func(t *testing.T) {
		config := Config{
			BucketName:      "test-bucket",
			CredentialsPath: "/nonexistent/path/key.json",
		}
		_, err := NewClient(ctx, config)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("missing bucket name", func(t *testing.T) {
		config := Config{
			BucketName:      "",
			CredentialsPath: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		}
		_, err := NewClient(ctx, config)
		assert.Error(t, err)
	})
}
