# GCS Client Implementation Plan

## Overview

This document outlines the implementation plan for the Google Cloud Storage (GCS) client layer in the Bina Marga Survey Photo Service. The GCS client is responsible for managing photo storage operations including signed URL generation, file existence checks, and file deletion.

**Target Package:** `internal/client/gcs/`
**Interface Location:** `internal/service/service.go` (lines 68-77)

## Goals

1. Implement the `GCSClient` interface defined in the service layer
2. Support two-phase upload workflow with signed URLs
3. Enable photo download and deletion operations
4. Provide comprehensive integration tests using real GCS bucket
5. Follow Go best practices and project conventions

## Architecture & Design Decisions

### Authentication Method: Service Account JSON Key

**Decision:** Use service account JSON key file for authentication.

**Rationale:**
- Explicit credential management suitable for both development and production
- Supports signed URL generation using the service account's private key
- No dependency on external metadata servers (works everywhere)
- Clear audit trail through service account permissions

**Security Considerations:**
- JSON key file must never be committed to version control
- File should have restrictive file permissions (0600)
- Environment variable points to file location, not the key content itself

### Signed URL Method: Private Key Signing

**Decision:** Use the service account's private key from JSON file for signing URLs.

**Implementation:**
```go
storage.SignedURLOptions{
    GoogleAccessID: serviceAccountEmail,
    PrivateKey:     privateKey,
    Method:         http.MethodPut, // or http.MethodGet
    Expires:        time.Now().Add(expiryDuration),
    ContentType:    contentType,
}
```

**Rationale:**
- Works with any service account that has `roles/storage.objectAdmin`
- No additional IAM permissions required beyond standard storage access
- Simpler than using IAM SignBlob API which requires additional permissions
- Suitable for the current service architecture

### Configuration: Environment Variables

**Required Variables:**

| Variable | Description | Example |
|----------|-------------|---------|
| `GCS_BUCKET_NAME` | Name of the GCS bucket | `bm-survey-photos-dev` |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to service account JSON | `/path/to/key.json` |

**Optional Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `GCS_TEST_PREFIX` | Prefix for test files | `test/` |
| `GCS_SIGNED_URL_EXPIRY_MINUTES` | Default signed URL expiry | `15` |
| `GCS_CONNECT_TIMEOUT_SECONDS` | Connection timeout | `30` |

## File Structure

```
internal/client/gcs/
├── client.go              # Main client implementation
├── client_test.go         # Unit tests (mocks, if needed later)
├── client_integration_test.go  # Integration tests with real GCS
├── config.go              # Configuration structures
├── errors.go              # GCS-specific error types
└── README.md              # Package documentation
```

## Implementation Details

### 1. Configuration (`config.go`)

```go
package gcs

// Config holds GCS client configuration
type Config struct {
    BucketName           string
    CredentialsPath      string
    TestPrefix           string
    SignedURLExpiryMins  int
    ConnectTimeoutSecs   int
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (Config, error)

// Validate validates the configuration
func (c Config) Validate() error
```

### 2. Error Types (`errors.go`)

```go
package gcs

import "errors"

var (
    ErrBucketNotFound     = errors.New("GCS bucket not found")
    ErrObjectNotFound     = errors.New("GCS object not found")
    ErrPermissionDenied   = errors.New("permission denied accessing GCS")
    ErrInvalidCredentials = errors.New("invalid service account credentials")
    ErrSignedURLFailed    = errors.New("failed to generate signed URL")
)
```

### 3. Client Implementation (`client.go`)

```go
package gcs

import (
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
)

// Client implements the service.GCSClient interface
type Client struct {
    client         *storage.Client
    bucket         *storage.BucketHandle
    bucketName     string
    serviceAccount *ServiceAccountInfo
}

// ServiceAccountInfo holds parsed service account details
type ServiceAccountInfo struct {
    ClientEmail string `json:"client_email"`
    PrivateKey  string `json:"private_key"`
    ProjectID   string `json:"project_id"`
}

// NewClient creates a new GCS client
func NewClient(ctx context.Context, config Config) (*Client, error)

// GenerateSignedURL generates a signed URL for uploading or downloading
func (c *Client) GenerateSignedURL(objectName string, contentType string, expiryMinutes int) (string, error)

// FileExists checks if a file exists in GCS
func (c *Client) FileExists(objectName string) (bool, error)

// DeleteFile deletes a file from GCS
func (c *Client) DeleteFile(objectName string) error

// Close closes the GCS client connection
func (c *Client) Close() error
```

### 4. Method Specifications

#### GenerateSignedURL

**Purpose:** Generate a time-limited signed URL for direct GCS access

**Parameters:**
- `objectName`: Full path within bucket (e.g., "photos/2026/NR-001/NR-001_2026_L1_a1b2c3d4.jpg")
- `contentType`: MIME type for upload URLs (e.g., "image/jpeg")
- `expiryMinutes`: URL validity duration in minutes

**Returns:**
- Signed URL string
- Error if signing fails

**Implementation Steps:**
1. Validate inputs (objectName not empty, expiry > 0)
2. Parse expiry duration
3. Build `storage.SignedURLOptions` with service account credentials
4. Call `storage.SignedURL()` to generate URL
5. Return signed URL or wrapped error

**Error Cases:**
- Invalid credentials
- Invalid object name
- Signing failure (permissions, network)

#### FileExists

**Purpose:** Check if a file exists in the GCS bucket

**Parameters:**
- `objectName`: Path to check

**Returns:**
- `true` if file exists
- `false` if file doesn't exist
- Error for unexpected failures

**Implementation Steps:**
1. Get `storage.ObjectHandle` for the object name
2. Call `Attrs()` to fetch object attributes
3. Return true if successful, false if not found error
4. Return error for other failures

**Error Cases:**
- Permission denied
- Network failures
- Invalid bucket configuration

#### DeleteFile

**Purpose:** Delete a file from GCS

**Parameters:**
- `objectName`: Path to delete

**Returns:**
- nil on success
- Error on failure

**Implementation Steps:**
1. Get `storage.ObjectHandle` for the object name
2. Call `Delete()` to remove the object
3. Return nil on success, wrapped error on failure

**Error Cases:**
- Object not found (may or may not be an error depending on use case)
- Permission denied
- Network failures

## Integration Testing Strategy

### Test Environment

**Bucket Structure:**
```
bm-survey-photos-dev/
├── photos/           # Production photos
│   ├── 2026/
│   └── ...
├── test/             # Test isolation directory
│   ├── integration-{timestamp}/
│   └── cleanup markers
└── ...
```

**Test Isolation:**
- All tests write to `test/` prefix only
- Each test run uses unique subdirectory with timestamp
- Tests clean up created files after completion
- Parallel tests use unique filenames to avoid collisions

### Test Cases

#### TestGenerateSignedURL
```go
func TestGenerateSignedURL(t *testing.T)
```
- Generate upload URL with valid parameters
- Verify URL format (contains storage.googleapis.com)
- Verify URL contains expected object name
- Test with different content types
- Test expiry time validation

#### TestGenerateSignedURLDownload
```go
func TestGenerateSignedURLDownload(t *testing.T)
```
- Generate download URL for existing object
- Verify URL is accessible via HTTP GET
- Verify URL expires after expiry time

#### TestFileExists
```go
func TestFileExists(t *testing.T)
```
- Upload test file via signed URL
- Verify FileExists returns true
- Delete file
- Verify FileExists returns false
- Test with non-existent object

#### TestDeleteFile
```go
func TestDeleteFile(t *testing.T)
```
- Upload test file
- Verify file exists
- Delete file
- Verify file no longer exists
- Test idempotent delete (deleting non-existent file)

#### TestUploadViaSignedURL
```go
func TestUploadViaSignedURL(t *testing.T)
```
- Generate signed upload URL
- Upload test image via HTTP PUT
- Verify upload succeeds (200 OK)
- Verify file exists in GCS
- Verify file content matches

#### TestEndToEndUploadConfirm
```go
func TestEndToEndUploadConfirm(t *testing.T)
```
- Simulate Phase 1: Generate signed URL
- Simulate Phase 2: Upload file to URL
- Simulate Phase 3: Verify file exists
- Simulate Phase 4: Delete file

### Test Prerequisites

**Environment Setup:**
```bash
# Required environment variables
export GCS_BUCKET_NAME="bm-survey-photos-dev"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/sidako-prod-1a52b69059a6.json"
export GCS_TEST_PREFIX="test/"
```

**Service Account Permissions Required:**
- `roles/storage.objectAdmin` (read/write/delete objects)
- Service account must be granted access to the bucket

### Test Execution

```bash
# Run all integration tests
go test -v ./internal/client/gcs/... -tags=integration

# Run specific test
go test -v ./internal/client/gcs/... -run TestGenerateSignedURL -tags=integration

# Run with timeout
go test -v ./internal/client/gcs/... -timeout=5m -tags=integration
```

## Dependencies

### New Dependencies to Add

```go
// go.mod additions
require (
    cloud.google.com/go/storage v1.50.0
    google.golang.org/api v0.224.0
)
```

### Install Command

```bash
go get cloud.google.com/go/storage
go get google.golang.org/api/option
```

## Implementation Phases

### Phase 1: Project Setup (30 minutes)
1. Create `internal/client/gcs/` directory
2. Add dependencies to go.mod
3. Create `config.go` with Config struct
4. Create `errors.go` with error definitions

### Phase 2: Core Client Implementation (60 minutes)
1. Implement `NewClient()` constructor
2. Implement `GenerateSignedURL()` method
3. Implement `FileExists()` method
4. Implement `DeleteFile()` method
5. Implement `Close()` method for cleanup

### Phase 3: Integration Tests (60 minutes)
1. Create `client_integration_test.go`
2. Implement test setup/teardown helpers
3. Write TestGenerateSignedURL
4. Write TestFileExists
5. Write TestDeleteFile
6. Write TestUploadViaSignedURL
7. Write TestEndToEndUploadConfirm

### Phase 4: Documentation (30 minutes)
1. Write package README.md
2. Add code comments for exported functions
3. Update main project documentation
4. Create usage examples

### Phase 5: Integration with Service Layer (30 minutes)
1. Update service initialization code
2. Add GCS client to dependency injection
3. Verify interface compliance
4. Test with actual service calls

## Security Considerations

### Credential Management

1. **JSON Key File:**
   - Store in secure location outside repository
   - File permissions: `chmod 600 key.json`
   - Never log or print key contents

2. **Environment Variables:**
   - Use `.env` file (already in .gitignore)
   - Validate path exists before initialization

3. **Signed URLs:**
   - Use minimum required expiry time (15 minutes default)
   - Validate content type matches expected format
   - Include bucket name in URL validation

### Access Control

1. **Service Account Permissions:**
   - Grant minimum required permissions
   - Use bucket-level IAM, not project-level when possible
   - Regular audit of service account usage

2. **Bucket Configuration:**
   - Enable uniform bucket-level access
   - Configure CORS for upload endpoints
   - Enable access logs for audit trail

## Error Handling Strategy

### Error Wrapping

All GCS errors should be wrapped with context:

```go
// Example error wrapping
if err != nil {
    return fmt.Errorf("failed to generate signed URL for %s: %w", objectName, err)
}
```

### Error Classification

| Error Type | GCS Error | Action |
|------------|-----------|--------|
| Not Found | `storage.ErrObjectNotExist` | Return specific error or false |
| Permission | `googleapi.Error` (403) | Return ErrPermissionDenied |
| Network | Context timeout | Return wrapped error with context |
| Invalid Creds | JSON parse error | Return ErrInvalidCredentials |

### Retry Logic

Consider implementing exponential backoff for:
- Network timeouts
- Rate limiting (429 errors)
- Temporary service unavailability

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/bina-marga/survey-photo/internal/client/gcs"
)

func main() {
    ctx := context.Background()
    
    config := gcs.Config{
        BucketName:      os.Getenv("GCS_BUCKET_NAME"),
        CredentialsPath: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
    }
    
    client, err := gcs.NewClient(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Generate upload URL
    signedURL, err := client.GenerateSignedURL(
        "photos/2026/NR-001/test.jpg",
        "image/jpeg",
        15,
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Signed URL:", signedURL)
}
```

### In Service Layer

```go
// internal/service/upload.go
func (s *UploadServiceImpl) GetSignedURL(...) {
    // ... validation code ...
    
    signedURL, err := s.gcsClient.GenerateSignedURL(
        gcsObjectName,
        req.FileMetadata.ContentType,
        15, // 15 minutes expiry
    )
    if err != nil {
        return nil, NewServiceError("STORAGE_ERROR", "Failed to generate signed URL", err)
    }
    
    // ... rest of method ...
}
```

## Success Criteria

The implementation is considered complete when:

1. ✅ All interface methods implemented and tested
2. ✅ Integration tests pass against real GCS bucket
3. ✅ All tests use `test/` prefix for isolation
4. ✅ Error handling follows project conventions
5. ✅ Code passes linting (`golangci-lint run`)
6. ✅ Documentation is complete
7. ✅ Service layer can successfully use the client

## Next Steps

1. Review this plan with stakeholders
2. Set up test bucket and service account permissions
3. Begin Phase 1 implementation
4. Run integration tests in development environment
5. Integrate with existing service layer
6. Update deployment documentation

---

**Document Version:** 1.0  
**Last Updated:** March 31, 2026  
**Author:** AI Assistant
