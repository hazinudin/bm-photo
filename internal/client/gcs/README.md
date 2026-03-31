# GCS Client Package

Google Cloud Storage client for the Bina Marga Survey Photo Service.

## Overview

This package provides a Go client for interacting with Google Cloud Storage (GCS) to support photo upload, download, and management operations. It implements the `GCSClient` interface defined in `internal/service/service.go`.

## Features

- **Signed URL Generation**: Create time-limited URLs for direct GCS uploads/downloads
- **File Operations**: Check file existence and delete files
- **Service Account Authentication**: Uses JSON key file for authentication
- **Context Support**: All operations support context for cancellation and timeouts

## Installation

Dependencies are managed via go modules:

```bash
go get cloud.google.com/go/storage
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/bina-marga/survey-photo/internal/client/gcs"
)

func main() {
    ctx := context.Background()
    
    // Load configuration from environment
    config, err := gcs.LoadConfigFromEnv()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create client
    client, err := gcs.NewClient(ctx, config)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Generate signed upload URL
    signedURL, err := client.GenerateSignedURL(
        "photos/2026/NR-001/test.jpg",
        "image/jpeg",
        15, // 15 minutes expiry
    )
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Signed URL:", signedURL)
}
```

### Configuration

Configuration is loaded from environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `GCS_BUCKET_NAME` | Yes | GCS bucket name |
| `GOOGLE_APPLICATION_CREDENTIALS` | Yes | Path to service account JSON key file |
| `GCS_TEST_PREFIX` | No | Prefix for test files (default: `test/` ) |
| `GCS_SIGNED_URL_EXPIRY_MINUTES` | No | Default signed URL expiry (default: 15) |
| `GCS_CONNECT_TIMEOUT_SECONDS` | No | Connection timeout (default: 30) |

### Service Account Setup

1. Create a service account in Google Cloud Console
2. Grant the following roles:
   - `roles/storage.objectAdmin` (read/write/delete objects)
3. Download the JSON key file
4. Set `GOOGLE_APPLICATION_CREDENTIALS` to the key file path

### Integration with Service Layer

```go
// In your service initialization
config := gcs.Config{
    BucketName:      os.Getenv("GCS_BUCKET_NAME"),
    CredentialsPath: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
}

gcsClient, err := gcs.NewClient(ctx, config)
if err != nil {
    log.Fatal(err)
}

// Use in services
uploadService := service.NewUploadService(photoRepo, pendingRepo, gcsClient, logger)
photoService := service.NewPhotoService(photoRepo, gcsClient, logger, auditSvc)
```

## Interface

The client implements the following interface:

```go
type GCSClient interface {
    // GenerateSignedURL creates a signed URL for upload or download
    GenerateSignedURL(objectName string, contentType string, expiryMinutes int) (string, error)
    
    // FileExists checks if a file exists in GCS
    FileExists(objectName string) (bool, error)
    
    // DeleteFile deletes a file from GCS
    DeleteFile(objectName string) error
}
```

## Error Handling

The package defines several error types:

- `ErrBucketNotFound`: Bucket does not exist or is not accessible
- `ErrObjectNotFound`: Object does not exist
- `ErrPermissionDenied`: Insufficient permissions
- `ErrInvalidCredentials`: Service account credentials are invalid
- `ErrSignedURLFailed`: Failed to generate signed URL

All errors are wrapped with context for better debugging.

## Testing

### Integration Tests

Integration tests run against a real GCS bucket and require proper credentials:

```bash
export GCS_BUCKET_NAME="bm-survey-photos-dev"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
export GCS_TEST_PREFIX="test/"

go test -v ./internal/client/gcs/... -tags=integration -timeout=5m
```

**Important:** Integration tests write to the `test/` prefix only. All test files are cleaned up after tests complete.

### Running Specific Tests

```bash
# Run specific test
go test -v ./internal/client/gcs/... -run TestGenerateSignedURL -tags=integration

# Run with timeout
go test -v ./internal/client/gcs/... -tags=integration -timeout=5m
```

## Architecture

### Signed URL Generation

Signed URLs allow clients to upload/download directly to/from GCS without going through the application server:

1. Client requests signed URL from application
2. Application generates URL using service account private key
3. Client uploads/downloads directly to GCS using the URL
4. URL expires after the specified duration

### Authentication

The client uses service account JSON key authentication:

1. Load JSON key file from configured path
2. Parse client email and private key
3. Use credentials to create GCS client
4. Use private key for signing URLs

## Security Considerations

1. **Credential Storage**: Never commit the JSON key file to version control
2. **File Permissions**: Set restrictive permissions on the key file (0600)
3. **URL Expiry**: Use minimum required expiry time (15 minutes recommended)
4. **Test Isolation**: All tests write to `test/` prefix to avoid polluting production data
5. **Service Account Permissions**: Grant minimum required permissions only

## Troubleshooting

### "invalid service account credentials" error

- Verify the JSON key file exists at the specified path
- Verify the file contains valid JSON with `client_email` and `private_key` fields
- Verify the service account has not been deleted or disabled

### "permission denied" error

- Verify the service account has `roles/storage.objectAdmin` role
- Verify the bucket exists and is accessible
- Check IAM permissions on the bucket

### Signed URL not working

- Verify the URL hasn't expired
- Verify the Content-Type header matches what was specified when generating the URL
- Check that the HTTP method matches (PUT for upload, GET for download)

## License

Part of the Bina Marga Survey Photo Service.
