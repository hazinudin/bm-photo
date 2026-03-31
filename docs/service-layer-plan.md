# Service Layer Implementation Plan

**Project:** Bina Marga Survey Photo Service  
**Version:** 1.0  
**Date:** March 31, 2026  
**Status:** In Progress

---

## Overview

This document outlines the comprehensive implementation plan for the Service Layer of the Bina Marga Survey Photo Service. The service layer acts as the orchestration layer between the REST/gRPC handlers and the repository layer, implementing business logic, validation, and workflow coordination.

## Current State

### Completed ✅
- **Domain Layer**: Value objects, entities, DTOs, errors, constants
- **Repository Layer**: PostgreSQL implementations with full CRUD operations
- **Service Layer**: Core interfaces and implementations created

### Pending ⏳
- Service layer unit tests
- GCS Client integration (for signed URL generation)
- Handler layer (REST API endpoints)
- Configuration management
- Main entry point

---

## Architecture

### Service Layer Structure

```
internal/service/
├── service.go          # Service interfaces (UploadService, PhotoService, etc.)
├── errors.go           # Service-specific errors
├── upload.go           # UploadService implementation
├── photo.go            # PhotoService implementation
├── auth.go             # AuthService implementation
├── audit.go            # AuditLogService implementation
└── *._test.go         # Unit tests (pending)
```

### Service Responsibilities

1. **UploadService**: Two-phase upload workflow orchestration
2. **PhotoService**: Photo catalog operations (CRUD, browse, search)
3. **AuthService**: API key validation and scope checking
4. **AuditLogService**: Operation logging for audit trail

### Dependencies

```
Handlers (REST/gRPC)
    ↓
Service Layer (this document)
    ↓
Repository Layer
    ↓
Database (PostgreSQL)
```

---

## Implementation Details

### 1. UploadService

**Purpose**: Orchestrate the two-phase upload workflow per PRD specification.

#### Phase 1: Get Signed Upload URL

**Method**: `GetSignedURL(ctx, req) (*GetSignedURLResponse, error)`

**Flow**:
1. Validate API key (via AuthService)
2. Validate request DTO (file metadata + photo attributes)
3. Check concurrent upload limit (max 10 pending per API key)
4. Generate identifiers:
   - photo_id: UUID v4
   - upload_token: UUID v4
   - gcs_object_name: `photos/{year}/{route_id}/{route_id}_{year}_{lane}_{shortuuid}.{ext}`
5. Create Photo entity using `entity.NewPhoto()`
6. Create PendingUpload record
7. Save both in database transaction
8. Generate signed URL via GCS client (15-minute expiry)
9. Log audit entry
10. Return upload token, signed URL, photo ID, expiry

**Error Handling**:
- `ErrUploadQuotaExceeded`: Too many pending uploads
- `ErrInvalidFileMetadata`: Invalid file size or content type
- `ErrInvalidCoordinates`: Latitude/longitude out of range
- `ErrInvalidLaneCode`: Invalid lane code format (L1-L10, R1-R10)

**Validation Rules**:
- File size: 1 byte to 10MB (10,485,760 bytes)
- Content type: image/jpeg or image/png only
- Coordinates: latitude [-90, 90], longitude [-180, 180]
- Lane code: regex `^[LR]\d{1,2}$` (L1-L10, R1-R10)
- STA value: optional, must be ≥ 0 if provided

#### Phase 2: Confirm Upload

**Method**: `ConfirmUpload(ctx, token) error`

**Flow**:
1. Validate API key
2. Lookup pending upload by token
3. Verify token exists and status is 'pending'
4. Verify token not expired (15-minute limit)
5. Verify API key matches token record
6. Get photo record by upload token
7. Verify file exists in GCS (HEAD request)
8. Mark pending upload as 'completed'
9. Log audit entry
10. Return success

**Error Handling**:
- `ErrTokenNotFound`: Token doesn't exist
- `ErrTokenExpired`: Token past expiry time
- `ErrTokenAlreadyUsed`: Token already marked completed
- `ErrFileNotFound`: File not found in GCS

#### GCS Object Name Generation

**Format**: `photos/{year}/{route_id}/{route_id}_{year}_{lane}_{shortuuid}.{ext}`

**Example**: `photos/2026/NR-001/NR-001_2026_L1_a1b2c3d4.jpg`

**Components**:
- Year: Current year (4 digits)
- Route ID: From photo attributes
- Lane: From photo attributes (e.g., L1, R3)
- Short UUID: 8-character alphanumeric
- Extension: From file format (jpg, png)

### 2. PhotoService

**Purpose**: Handle photo catalog operations.

#### Core Operations

**GetByID**:
- Retrieve photo by ID
- Return error if photo is soft-deleted
- Map repository errors to service errors

**Browse**:
- Filter by: route_id (required), sta_start, sta_end, lane
- Pagination: page, per_page (default: 20, max: 100)
- Return paginated list with total count

**Search**:
- Advanced search with multiple filters:
  - route_ids: array of route IDs
  - sta_ranges: array of {start, end} pairs
  - lanes: array of lane codes
  - date_start/date_end: upload date range
  - tags: array of tags (AND logic)
- Pagination support

**Update**:
- Update mutable fields: description, tags, lane_code
- Validate lane_code format if provided
- Prevent updates to deleted photos
- Log audit entry

**Delete**:
- Soft delete (default): Mark as deleted, keep files
- Hard delete (admin only): Remove from DB and GCS
- Log audit entry

### 3. AuthService

**Purpose**: API key authentication and authorization.

#### API Key Validation

**Method**: `ValidateAPIKey(ctx, key string) (*APIKey, error)`

**Flow**:
1. Hash the provided API key (SHA-256)
2. Lookup in database by hash
3. Check if key is active
4. Check if key is expired (if expiry set)
5. Update last_used_at timestamp
6. Return API key entity

**Error Handling**:
- `ErrAPIKeyInvalid`: Key not found
- `ErrAPIKeyInactive`: Key revoked
- `ErrAPIKeyExpired`: Key past expiry date

#### Scope Checking

**Method**: `CheckScope(key *APIKey, scope string) error`

**Scopes**:
- `read`: Browse and download photos
- `write`: Upload and update photos
- `admin`: Delete photos and manage keys

**Helpers**:
- `CheckReadScope(key)`: Verify 'read' scope
- `CheckWriteScope(key)`: Verify 'write' scope
- `CheckAdminScope(key)`: Verify 'admin' scope

### 4. AuditLogService

**Purpose**: Immutable logging of all photo operations.

#### Log Operations

**LogUploadRequest**:
- Operation: 'upload_signed_url'
- Details: filename, content_type, file_size, route_id, lane_code

**LogUploadConfirm**:
- Operation: 'upload_confirm'
- Details: upload_token, confirmation timestamp

**LogPhotoUpdate**:
- Operation: 'photo_update'
- Details: changed_fields (description, tags, lane_code)

**LogPhotoDelete**:
- Operation: 'photo_delete'
- Details: deletion_type (soft/hard), deleted_by

**LogPhotoRestore**:
- Operation: 'photo_restore'
- Details: restored_by, restoration timestamp

**LogPhotoDownload**:
- Operation: 'photo_download'
- Details: download_type (original/thumbnail), size

---

## Testing Strategy

### Unit Tests

**Test File Pattern**: `*_test.go` in same directory as implementation

**Coverage Requirements**:
- All public methods must have tests
- Error paths must be tested
- Validation logic must be tested
- Transaction rollback scenarios

**Test Structure**:
```go
func TestUploadService_GetSignedURL_ValidRequest(t *testing.T) {
    // Setup mocks
    // Call method
    // Assert results
}

func TestUploadService_GetSignedURL_QuotaExceeded(t *testing.T) {
    // Test upload limit enforcement
}
```

**Mocking Strategy**:
- Use `mockgen` for repository interfaces
- Use `testify/mock` for assertions
- Mock GCS client for upload tests

### Integration Tests

**Scope**: Service layer with real repositories (PostgreSQL container)

**Test Scenarios**:
- Complete two-phase upload flow
- Photo CRUD operations
- Concurrent upload limit enforcement
- Token expiration handling

---

## Error Handling

### ServiceError Structure

```go
type ServiceError struct {
    Code    string // Machine-readable code
    Message string // Human-readable message
    Err     error  // Wrapped error for chain
}
```

### Error Mapping

| Repository Error | Service Error | HTTP Status |
|-----------------|---------------|-------------|
| ErrPhotoNotFound | ErrPhotoNotFound | 404 Not Found |
| ErrTokenNotFound | ErrInvalidToken | 404 Not Found |
| ErrTokenExpired | ErrTokenExpired | 410 Gone |
| ErrUploadQuotaExceeded | ErrUploadQuotaExceeded | 429 Too Many Requests |

---

## Configuration

### Service Configuration

```yaml
upload:
  max_file_size_mb: 10
  signed_url_expiry_minutes: 15
  max_pending_uploads_per_key: 10

gcs:
  bucket_name: "bm-survey-photos"
  signed_url_expiry: "15m"

rate_limits:
  signed_url_per_minute: 10
  confirm_per_minute: 10
  browse_per_minute: 100
```

---

## Next Steps

### Immediate (This Sprint)
1. ✅ Implement UploadService
2. ✅ Implement PhotoService
3. ✅ Implement AuthService
4. ✅ Implement AuditLogService
5. ⏳ Write comprehensive unit tests
6. ⏳ Write integration tests with PostgreSQL

### Next Sprint
7. ⏳ Implement GCS Client (signed URL generation)
8. ⏳ Implement REST handlers
9. ⏳ Implement middleware (auth, rate limiting)
10. ⏳ Create main entry point

### Future Enhancements
11. ⏳ Add LRS client integration (STA interpolation)
12. ⏳ Add gRPC handlers for catalog browsing
13. ⏳ Add EXIF extraction service (deferred feature)
14. ⏳ Add thumbnail generation service (deferred feature)

---

## References

- **PRD**: `/PRD.md` - Complete API specification
- **Domain Layer Plan**: `/docs/domain-layer-plan.md` - Entity definitions
- **Repository Layer Plan**: `/docs/repository-layer-plan.md` - Database operations
- **AGENTS.md**: `/AGENTS.md` - Code style guidelines

---

## Success Criteria

- [x] All service interfaces defined
- [x] UploadService implements two-phase upload
- [x] PhotoService implements CRUD operations
- [x] AuthService validates API keys and scopes
- [x] AuditLogService logs all operations
- [ ] Unit tests > 80% coverage
- [ ] Integration tests pass
- [ ] No lint errors
- [ ] All error cases handled
