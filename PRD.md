# Product Requirements Document (PRD)
# Bina Marga Survey Photo Service

**Version:** 1.7  
**Date:** April 9, 2026  
**Status:** In Progress - GCS Object Name Fix + UUID v7 Migration  

---

## Implementation Status

### Completed Components

| Component | Status | Location | Notes |
|-----------|--------|----------|-------|
| Domain Layer | ✅ Complete | `internal/model/` | VOs, Entities, DTOs, Errors, Constants |
| Value Objects | ✅ Complete | `internal/model/vo/` | PhotoID, UploadToken, Coordinates, FileFormat, UploadStatus, STASource |
| Photo Entity | ✅ Complete | `internal/model/entity/photo.go` | Simplified aggregate root |
| REST DTOs | ✅ Complete | `internal/model/dto/rest/` | Upload, Photo, Browse DTOs with validation |
| Domain Errors | ✅ Complete | `internal/model/error.go` | All domain errors defined |
| Domain Constants | ✅ Complete | `internal/model/constants.go` | File limits, pagination, rate limits |
| Unit Tests | ✅ Complete | `internal/model/**/*_test.go` | Tests for VOs, entities, DTOs |
| Repository Layer | ✅ Complete | `internal/repository/` | Interfaces, PostgreSQL implementations |
| PhotoRepository | ✅ Complete | `internal/repository/postgres/photo.go` | Full CRUD, Browse, Search |
| PendingUploadRepository | ✅ Complete | `internal/repository/postgres/pending_upload.go` | Token lifecycle management |
| APIKeyRepository | ✅ Complete | `internal/repository/postgres/api_key.go` | Hash-based lookup, scopes |
| AuditLogRepository | ✅ Complete | `internal/repository/postgres/audit_log.go` | Immutable logging |
| Repository Integration Tests | ✅ Complete | `internal/repository/postgres/*_test.go` | 30+ tests |
| Database Migrations | ✅ Complete | `migrations/` | Initial + simplification migration |
| pgx v5 Driver | ✅ Complete | `go.mod` | PostgreSQL driver added |
| Service Layer | ✅ Complete | `internal/service/` | PhotoService, UploadService, AuthService, AuditService |
| Retry Endpoint Logic | ⏳ Pending | `internal/service/` | GetNewSignedURL method for retry uploads |

### Pending Components

| Component | Status | Priority | Notes |
|-----------|--------|----------|-------|
| Handler Layer | ❌ Not Started | High | REST handlers for all endpoints |
| gRPC/Proto | ❌ Not Started | Medium | Catalog browsing service definition |
| GCS Client | ❌ Not Started | High | Signed URL, upload/download |
| LRS Client | ❌ Not Started | High | gRPC client for STA calculation |
| Main Entry Point | ❌ Not Started | High | `cmd/server/main.go` |
| Configuration | ❌ Not Started | High | YAML config loading |

### Bug Fixes & Known Issues

| Issue | Date | Status | Fix |
|-------|------|--------|-----|
| **GCS Object Name Collision (UUID v7 Migration)** | Apr 9, 2026 | ✅ Fixed | `internal/service/upload.go` - Changed `generateGCSObjectName` to use the real photo ID (UUID v7, 36 chars) instead of generating a new random short UUID. This prevented multiple photos from overwriting each other in GCS when using UUID v7 (which has a different byte layout than v4, causing apparent collisions when only first 8 chars were used). Added `SetGCSObjectName()` method to `Photo` entity to update GCS object name after photo ID is generated. |
| **UUID Version** | Apr 9, 2026 | ✅ Migrated | Photo IDs now use UUID v7 (time-ordered) instead of UUID v4. This maintains sortability by creation time while being globally unique. |

---

## Executive Summary

The Bina Marga Survey Photo Service is a photo catalog system designed to manage survey photographs of national routes maintained by Bina Marga (Indonesian Directorate General of Highways). The service provides an abstraction layer over photos stored in Google Cloud Storage, enabling users to upload, organize, and browse survey photos with geospatial metadata including route information, lane numbers, coordinates, and Linear Referencing System (LRS) station values.

**Key Objectives:**
- Centralized photo catalog for national route survey data
- Geospatial organization using Linear Referencing System (LRS)
- Integration with existing LRS microservice for STA calculation
- Support for external client applications to upload and query photos

---

## 1. Product Overview

### 1.1 Purpose

This service serves as the authoritative catalog for survey photographs of Indonesian national routes, providing:
- Photo upload with rich metadata attributes
- Geospatial organization via Linear Referencing System
- Efficient browsing and querying capabilities
- Integration with existing Bina Marga infrastructure

### 1.2 Scope

**In Scope (MVP):**
- Photo upload with metadata (route, lane, coordinates, STA) - REST API only
- Photo browsing/search by route, STA, and lane - REST API + gRPC API
- Google Cloud Storage integration
- REST API for all client operations
- gRPC API for internal microservice catalog browsing
- API Key authentication

**Out of Scope (Future Phases):**
- Automatic EXIF metadata extraction
- Thumbnail generation for preview
- Integration with existing LRS microservice (gRPC) for STA calculation
- Web-based user interface
- Mobile applications
- Machine learning-based photo analysis
- Real-time photo streaming
- Route condition analysis

### 1.3 Success Metrics

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Upload success rate | ≥99.5% | Monitoring dashboard |
| API response time (p95) | <500ms | Prometheus metrics |
| Upload throughput | 100-1000 photos/day | Application logs |
| LRS interpolation accuracy | ±5 meters | Comparison with manual measurements |
| Service availability | ≥99% | Uptime monitoring |

---

## 2. Functional Requirements

### 2.1 Photo Upload

#### 2.1.1 Two-Phase Upload Architecture

The photo upload process uses a two-phase approach to optimize for performance and scalability:

**Phase 1: Request Signed URL with Full Metadata (Client-Side)**
- Client provides ALL photo attributes upfront (file metadata + photo metadata)
- Backend validates all data, creates photo record, and generates signed URL
- Returns signed URL, upload token, and photo ID to client

**Phase 2: Confirm Upload Completion (Client-Side)**
- Client confirms successful GCS upload
- Backend verifies file exists in GCS
- Marks upload as completed (photo already has all metadata)

**Key Design Decision:** Photo attributes (route_id, lane_code, coordinates, etc.) are provided in Phase 1, not Phase 2. This simplifies the architecture and allows all validation to happen upfront. Phase 2 is purely a confirmation step.

**Deferred Features (Future Phases):**
- EXIF metadata extraction
- Thumbnail generation
- LRS integration for STA interpolation
- Post-upload processing workflow

####2.1.2 Upload Endpoints

**Endpoint 1: Get Signed Upload URL (Initial Upload)**
- **Priority:** Must Have
- **API Method:** POST /api/v1/photos/upload-url
- **Authentication:** API Key required
- **Purpose:** Generate a signed URL for direct client upload to GCS with full photo metadata. Creates a new photo record.

**Endpoint 2: Get New Signed URL (Retry Upload)**
- **Priority:** Must Have
- **API Method:** POST /api/v1/photos/{photo_id}/new-signed-url
- **Authentication:** API Key required
- **Purpose:** Generate a new signed URL for an existing pending photo. Used when GCS upload fails and client needs to retry with the same photo record.

**Endpoint 3: Confirm Upload**
- **Priority:** Must Have
- **API Method:** POST /api/v1/photos/confirm
- **Authentication:** API Key required
- **Purpose:** Confirm successful GCS upload

#### 2.1.3 Photo Attributes

| Attribute | Type | Required | Source | Endpoint | Description |
|-----------|------|----------|--------|----------|-------------|
| file_metadata | object | Yes | Client | upload-url | File metadata for signing |
| file_metadata.filename | string | Yes | Client | upload-url | Original filename |
| file_metadata.content_type | string | Yes | Client | upload-url | MIME type (image/jpeg or image/png) |
| file_metadata.file_size | integer | Yes | Client | upload-url | File size in bytes (max 10MB) |
| photo_attributes | object | Yes | Client | upload-url | Photo metadata |
| photo_attributes.route_id | string | Yes | Client | upload-url | Identifier for national route |
| photo_attributes.lane_code | string | Yes | Client | upload-url | Lane identifier (L1-L10 for left lanes, R1-R10 for right lanes) |
| photo_attributes.latitude | decimal | Yes | Client | upload-url | Latitude in decimal degrees (EPSG:4326) |
| photo_attributes.longitude | decimal | Yes | Client | upload-url | Longitude in decimal degrees (EPSG:4326) |
| photo_attributes.sta_value | decimal | No | Client | upload-url | Station value along route (optional) |
| photo_attributes.description | string | No | Client | upload-url | Optional photo description |
| photo_attributes.tags | array | No | Client | upload-url | Optional tags for categorization |
| upload_token | string | Yes | Client | confirm | Token from signed URL creation |

**Note:** LRS integration for STA calculation is deferred to a future phase. If `sta_value` is not provided, it can be calculated later by other services when coordinate data is available.

#### 2.1.4 Upload Workflow

##### 2.1.4.1 Initial Upload Flow

```
Phase 1: Get Signed Upload URL (with full metadata)
─────────────────────────────────────────────────────────┐
Client sends:                                            │
    ├─ file_metadata (filename, content_type, file_size) │
    └─ photo_attributes (route_id, lane_code)            │
    ↓                                                    │
Validate API Key                                         │
    ↓                                                    │
Validate file metadata                                   │
    ├─ Check file size (max 10MB)                        │
    ├─ Check content type (JPEG/PNG)                     │
    └─ Validate filename format                          │
    ↓                                                    │
Validate photo attributes                                │
    ├─ Validate route_id format                          │
    ├─ Validate lane_code format (L1-L10, R1-R10)        │
    ├─ Validate coordinates (lat/lon ranges)             │ 
    └─ Validate sta_value if provided                    │
    ↓                                                    │
Check concurrent upload limit for API key                │
    └─ Max 10 pending uploads per API key                │
    ↓                                                    │
Generate identifiers and GCS object name                 │
    ├─ Create unique photo_id (UUID v7)                 │
    ├─ Generate GCS object name:                         │
    │   photos/{year}/{route_id}/{route_id}_{year}_      │
    │   {lane}_{photo_id}.{ext}                          │
    │   Note: Full photo_id (36 chars) is used in GCS    │
    │   object name (not shortuuid). This ensures       │
    │   uniqueness especially after UUID v4→v7 migration.│
    ├─ Create upload_token (UUID)                        │
    └─ Create signed URL for that object path (15 min)   │
    ↓                                                    │
Create photo record in database                          │
    ├─ photo_id, route_id, lane_code                     │
    ├─ latitude, longitude, sta_value                    │
    ├─ gcs_object_name, file_format, file_size           │
    └─ upload_status = 'pending'                         │
    ↓                                                    │
Create pending_upload record                             │
    ├─ upload_token, photo_id                            │
    ├─ api_key_id, expires_at                            │
    └─ status = 'pending'                                │
    ↓                                                    │
Return to client                                         │
    ├─ signed_url (for uploading to GCS)                 │
    ├─ upload_token (for confirmation request)           │
    └─ photo_id (for client reference)                   │
─────────────────────────────────────────────────────────┘
```

##### 2.1.4.2 Retry Upload Flow (When GCS Upload Fails)

```
Retry: Get New Signed URL for Existing Photo
─────────────────────────────────────────────────────────┐
Prerequisites:                                           │
    ├─ Previous upload to GCS failed (network, timeout)  │
    ├─ Client has photo_id from initial request          │
    └─ Photo status is still 'pending'                   │
    ↓                                                    │
Client sends:                                            │
    POST /photos/{photo_id}/new-signed-url               │
    ↓                                                    │
Validate API Key                                         │
    ↓                                                    │
Validate photo_id exists and is pending                  │
    ├─ Check photo record exists                         │
    ├─ Verify photo.upload_status = 'pending'            │
    └─ Verify requesting API key matches photo creator   │
    ↓                                                    │
Invalidate previous tokens for this photo                │
    ├─ Mark existing pending uploads as 'expired'        │
    └─ Only most recent token remains valid              │
    ↓                                                    │
Generate new upload resources                            │
    ├─ Create new upload_token (UUID)                    │
    ├─ Generate new signed URL (15 min expiry)           │
    │   Note: GCS object name stays the same             │
    └─ Update photo.updated_at timestamp                 │
    ↓                                                    │
Create new pending_upload record                         │
    ├─ upload_token, photo_id                            │
    ├─ api_key_id, expires_at (+15 min)                  │
    └─ status = 'pending'                                │
    ↓                                                    │
Return to client                                         │
    ├─ signed_url (new URL for retry)                    │
    ├─ upload_token (new token for confirmation)         │
    └─ photo_id (same as before)                         │
    ↓                                                    │
Client uploads to GCS using new signed_url               │
    ↓                                                    │
Client confirms with new upload_token                    │
─────────────────────────────────────────────────────────┘
```

##### 2.1.4.3 Confirm Upload Flow

```
Phase 2: Confirm Upload
─────────────────────────────────────────────────────────────────┐
Client sends:                                            │
    └─ upload_token                                      │
    ↓                                                    │
Validate API Key                                         │
    ↓                                                    │
Validate upload token                                    │
    ├─ Look up pending_upload by upload_token            │
    ├─ Check token exists and not expired                │
    ├─ Verify token status is 'pending'                 │
    └─ Verify API key matches token record               │
    ↓                                                    │
Verify file exists in GCS                                │
    └─ Use gcs_object_name from photo record            │
    ↓                                                    │
Mark pending_upload status as 'completed'                │
    ↓                                                    │
Update photo record                                      │
    ├─ upload_status = 'completed'                       │
    └─ uploaded_at = CURRENT_TIMESTAMP                   │
    ↓                                                    │
Return confirmation to client                            │
    ├─ photo_id                                          │
    └─ message: "Upload confirmed"                       │
─────────────────────────────────────────────────────────────────┘

Note: EXIF extraction, thumbnail generation, and LRS integration
are deferred to future phases. These features will be triggered
by other services when coordinate data becomes available.
```

#### 2.1.5 Error Handling

**Phase 1 (Signed URL) Errors:**

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| INVALID_API_KEY | API key missing or invalid | 401 Unauthorized |
| INVALID_FILE_METADATA | Missing or invalid file metadata | 400 Bad Request |
| FILE_TOO_LARGE | File exceeds 10MB limit | 413 Payload Too Large |
| UNSUPPORTED_FORMAT | Content type not JPEG or PNG | 400 Bad Request |
| UPLOAD_QUOTA_EXCEEDED | Too many pending uploads for API key | 429 Too Many Requests |
| STORAGE_ERROR | Failed to generate signed URL | 500 Internal Server Error |
| INVALID_COORDINATES | Coordinates outside valid range | 400 Bad Request |
| INVALID_ROUTE_ID | Invalid route ID format | 400 Bad Request |
| INVALID_LANE_CODE | Invalid lane code format | 400 Bad Request |

**Client-Side GCS Upload Errors:**

| Error Code | Description | Resolution |
|------------|-------------|------------|
| UnsignedUpload | Attempting upload without valid signed URL | Obtain new signed URL |
| ExpiredURL | Signed URL has expired (15 min timeout) | Request new signed URL |
| CORS | Pre-flight request blocked | Configure GCS bucket CORS |
| NetworkError | Upload interrupted | Request new signed URL |

**Retry Upload (New Signed URL) Errors:**

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| INVALID_API_KEY | API key missing or invalid | 401 Unauthorized |
| PHOTO_NOT_FOUND | Photo ID does not exist | 404 Not Found |
| PHOTO_ALREADY_COMPLETED | Photo has already been uploaded and confirmed | 409 Conflict |
| PHOTO_NOT_OWNED | Photo was created by a different API key | 403 Forbidden |
| RETRY_LIMIT_EXCEEDED | Maximum retry attempts (5) exceeded for this photo | 429 Too Many Requests |

**Phase 2 (Confirm Upload) Errors:**

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| INVALID_API_KEY | API key missing or invalid | 401 Unauthorized |
| INVALID_UPLOAD_TOKEN | Token missing, expired, or already used | 400 Bad Request |
| TOKEN_NOT_FOUND | Upload token does not exist | 404 Not Found |
| TOKEN_ALREADY_USED | Token has already been used | 409 Conflict |
| TOKEN_EXPIRED | Token has expired (15 minute timeout) | 410 Gone |
| FILE_NOT_FOUND | Photo file not found in GCS at expected location | 404 Not Found |

#### 2.1.6 Upload Token Lifecycle

**Token States:**
- `pending` → Initial state when signed URL is generated
- `completed` → Marked after successful upload confirmation
- `expired` → Marked after 15-minute timeout

**Token Enforcement:**
- Backend validates token state before processing confirmation
- Attempting to reuse a token results in TOKEN_ALREADY_USED error
- Token expiration prevents indefinite pending uploads
- **New tokens invalidate old ones**: When requesting a new signed URL for retry, previous tokens for the same photo are marked as 'expired'

**Client-Side Retry Logic:**
- If GCS upload fails (network error, timeout), request a NEW signed URL using `POST /photos/{photo_id}/new-signed-url`
- Use the same `photo_id` from the initial upload request
- Do NOT retry with the same signed URL after a failed upload
- If confirm request fails, token may still be valid for retry
- Maximum 5 retry attempts per photo (prevents abuse)

**Cleanup and Expiration:**
- Expired tokens are marked as 'expired' (background job runs hourly)
- **GCS files are NOT auto-deleted** when tokens expire (prevents race conditions)
- Photos associated with expired tokens remain in database for manual review
- Deferred processing features (EXIF, thumbnails, LRS) are not triggered for expired uploads
- Admin cleanup endpoint available to remove orphaned photos after 24+ hours

**Retry Flow with Dedicated Endpoint:**
```
1. Client requests signed URL with metadata
   POST /upload-url
   Response: {photo_id: "ABC", token: "token-1", signed_url: "url-1"}

2. Client attempts upload to GCS using signed_url
   PUT url-1 (binary data)
   Result: ❌ Network timeout

3. Client requests NEW signed URL for same photo
   POST /photos/ABC/new-signed-url
   Backend: Invalidates token-1, creates token-2
   Response: {photo_id: "ABC", token: "token-2", signed_url: "url-2"}

4. Client uploads to GCS using new signed_url
   PUT url-2 (binary data)
   Result: ✅ Success

5. Client confirms upload
   POST /confirm {token: "token-2"}
   Response: {photo_id: "ABC", message: "Upload confirmed"}

6. Token-1 remains in database as 'expired' for audit trail
```

### 2.2 Photo Browsing and Search

#### 2.2.1 Browse by Route
- **Priority:** Must Have
- **API Method:** GET /api/v1/photos?route_id={route_id}
- **Description:** Retrieve all photos for a specific route
- **Query Parameters:**
  - route_id (required): string
  - page (optional): integer, default 1
  - per_page (optional): integer, default 20, max 100
- **Response:** Paginated list of photo metadata

#### 2.2.2 Browse by Route and STA Range
- **Priority:** Must Have
- **API Method:** GET /api/v1/photos?route_id={route_id}&sta_start={start}&sta_end={end}
- **Description:** Retrieve photos within a specific STA range on a route
- **Query Parameters:**
  - route_id (required): string
  - sta_start (required): decimal
  - sta_end (required): decimal
  - page (optional): integer
  - per_page (optional): integer
- **Response:** Paginated list of photo metadata within STA range

#### 2.2.3 Browse by Route, STA, and Lane
- **Priority:** Must Have
- **API Method:** GET /api/v1/photos?route_id={route_id}&sta_start={start}&sta_end={end}&lane={lane}
- **Description:** Retrieve photos for specific route, STA range, and lane
- **Query Parameters:**
  - route_id (required): string
  - sta_start (required): decimal
  - sta_end (required): decimal
  - lane (required): integer
  - page (optional): integer
  - per_page (optional): integer
- **Response:** Paginated list of photo metadata

#### 2.2.4 Get Single Photo Metadata
- **Priority:** Must Have
- **API Method:** GET /api/v1/photos/{photo_id}
- **Description:** Retrieve complete metadata for a specific photo
- **Response:** Full photo metadata including EXIF data, URLs

#### 2.2.5 Download Photo
- **Priority:** Must Have
- **API Method:** GET /api/v1/photos/{photo_id}/download
- **Description:** Download original photo file
- **Response:** Binary file stream with appropriate Content-Type
- **Headers:**
  - Content-Type: image/jpeg or image/png
  - Content-Disposition: attachment; filename="{photo_id}.jpg"

#### 2.2.6 Download Thumbnail
- **Priority:** Must Have
- **API Method:** GET /api/v1/photos/{photo_id}/thumbnail?size={small|medium|large}
- **Description:** Download pre-generated thumbnail
- **Response:** Binary thumbnail file

#### 2.2.7 Advanced Search
- **Priority:** Should Have
- **API Method:** POST /api/v1/photos/search
- **Description:** Complex search with multiple filters
- **Request Body:** JSON with search criteria
- **Filters:**
  - route_ids (array of strings)
  - sta_ranges (array of start/end pairs)
  - lanes (array of integers)
  - date_range (start_date, end_date)
  - bounding_box (lat_min, lat_max, lon_min, lon_max)
  - tags (array of strings)
  - has_exif_gps (boolean)

### 2.3 Photo Management

#### 2.3.1 Update Photo Metadata
- **Priority:** Should Have
- **API Method:** PATCH /api/v1/photos/{photo_id}
- **Description:** Update metadata attributes (not the photo file itself)
- **Editable Fields:** description, tags, lane_code, latitude, longitude, sta_value
- **Authentication:** API Key with write permissions
- **Note:** Coordinates (latitude/longitude) must be provided together. When sta_value is updated, sta_source is set to user_provided.

#### 2.3.2 Delete Photo
- **Priority:** Must Have
- **API Method:** DELETE /api/v1/photos/{photo_id}
- **Description:** Soft delete (mark as deleted) or hard delete
- **Authentication:** API Key with admin permissions
- **Behavior:**
  - Soft delete: Mark record as deleted, retain files
  - Hard delete: Remove from database and delete from GCS
- **Default:** Soft delete

#### 2.3.3 Bulk Delete
- **Priority:** Should Have
- **API Method:** POST /api/v1/photos/bulk-delete
- **Description:** Delete multiple photos by criteria
- **Request Body:** Filter criteria (route, date range, etc.)

---

## 3. Integration Requirements

### 3.1 Linear Referencing System (LRS) Service Integration

#### 3.1.1 LRS Service Interface
- **Integration Type:** gRPC (internal)
- **Service Definition:** To be provided by existing LRS microservice team

#### 3.1.2 Required Operations

**Operation: Interpolate STA from Coordinates**
```
Request:
  - route_id: string
  - latitude: decimal
  - longitude: decimal

Response:
  - sta_value: decimal
  - confidence_score: decimal (0.0-1.0)
  - interpolated: boolean

Error Cases:
  - Route not found
  - Coordinates outside route corridor
  - Service unavailable
```

**Operation: Validate STA**
```
Request:
  - route_id: string
  - latitude: decimal
  - longitude: decimal
  - sta_value: decimal

Response:
  - valid: boolean
  - deviation: decimal (meters)
  - corrected_sta: decimal (if invalid)
```

#### 3.1.3 LRS Service Contract
- **To Be Defined:** Need LRS service API specification from Bina Marga team
- **Information Needed:**
  - gRPC service definition (.proto file)
  - Host and port for LRS service
  - Authentication mechanism
  - Rate limits and quotas
  - Error codes and meanings

### 3.2 Google Cloud Storage Integration

#### 3.2.1 Storage Bucket Configuration
- **Bucket Name:** bm-survey-photos-{environment}
- **Bucket Structure:**
  ```
  /photos/
    /{year}/
      /{route_id}/
        /{route_id}_{year}_{lane}_{photo_id}.{ext}           # Original photo
        /{route_id}_{year}_{lane}_{photo_id}_small.{ext}     # Small thumbnail (150x150)
        /{route_id}_{year}_{lane}_{photo_id}_medium.{ext}    # Medium thumbnail (400x400)
        /{route_id}_{year}_{lane}_{photo_id}_large.{ext}     # Large thumbnail (800x800)
  ```
  
  Example:
  ```
  /photos/
    /2026/
      /NR-001/
        /NR-001_2026_L1_550e8400-e29b-41d4-a716-446655440000.jpg         # Original
        /NR-001_2026_L1_550e8400-e29b-41d4-a716-446655440000_small.jpg    # Small thumbnail
        /NR-001_2026_L1_550e8400-e29b-41d4-a716-446655440000_medium.jpg   # Medium thumbnail
        /NR-001_2026_L1_550e8400-e29b-41d4-a716-446655440000_large.jpg    # Large thumbnail
  ```

#### 3.2.2 Storage Operations
- **Upload:** Store original photo to GCS
- **Generate Thumbnails:** Create multiple thumbnail sizes
- **Delete:** Remove photo and thumbnails from GCS
- **Signed URLs:** Generate time-limited download URLs

#### 3.2.3 IAM and Permissions
- Service account with Storage Object Admin role
- Signed URL generation for secure downloads
- Bucket-level IAM policies

---

## 4. Non-Functional Requirements

### 4.1 Performance Requirements

| Metric | Requirement | Notes |
|--------|-------------|-------|
| API Response Time (p50) | <200ms | Typical requests |
| API Response Time (p95) | <500ms | Under normal load |
| API Response Time (p99) | <2000ms | Under peak load |
| Upload Throughput | 100-1,000 photos/day | MVP target |
| Concurrent Users | 10-50 simultaneous users | MVP target |
| Database Query Time (p95) | <100ms | For catalog queries |
| Thumbnail Generation | <5 seconds per photo | Background process |

### 4.2 Scalability Requirements

- **Horizontal Scaling:** Service should support horizontal scaling via load balancer
- **Database Connection Pooling:** pgx pool with configurable limits
- **Connection Pool Size:** Minimum 10, maximum 50 connections
- **Background Job Queue:** Async thumbnail generation
- **Caching:** Consider Redis for frequently accessed metadata

### 4.3 Availability Requirements

- **Service Availability:** ≥99% uptime
- **Planned Maintenance Windows:** Weekend nights (WIB timezone)
- **Failure Recovery:** Automatic restart on VM failure
- **Database Backups:** Daily backups with 30-day retention

### 4.4 Security Requirements

#### 4.4.1 Authentication
- **Method:** API Key
- **Key Format:** UUID v4
- **Key Storage:** Encrypted hash in database
- **Key Rotation:** Support key rotation without downtime

#### 4.4.2 Authorization
- **API Key Scopes:**
  - read: Read photos and metadata
  - write: Upload and update photos
  - admin: Delete photos and manage keys
- **Key Management:** Admin endpoints for key generation and revocation

#### 4.4.3 Data Security
- **In Transit:** TLS 1.3 for all API communication
- **At Rest:** Photos encrypted in GCS (Google-managed keys)
- **Database:** PostgreSQL encrypted at rest
- **Sensitive Data:** EXIF data may contain personal information (handle per policy)

#### 4.4.4 API Security
- **Rate Limiting:** 100 requests per minute per API key
- **Input Validation:** Sanitize all inputs, prevent SQL injection
- **File Validation:** Validate MIME types, not just extensions
- **Virus Scanning:** Consider scanning uploaded files

### 4.5 Reliability Requirements

- **Error Handling:** Graceful degradation when LRS service is unavailable
- **Retry Logic:** Exponential backoff for LRS service calls
- **Circuit Breaker:** Fail fast when LRS service is down
- **Idempotency:** Support idempotent photo uploads

### 4.6 Maintainability Requirements

- **Code Quality:** Follow Go best practices and idioms
- **Documentation:** API documentation using OpenAPI 3.0 specification
- **Logging:** Structured logging (JSON format) with configurable levels
- **Monitoring:** Prometheus metrics exposed for all operations
- **Health Checks:** /health endpoint for load balancer health checks

---

## 5. Technical Architecture

### 5.1 Technology Stack

| Component | Technology | Version | Notes |
|-----------|-----------|---------|-------|
| Language | Golang | 1.22+ | Primary service implementation |
| Web Framework | Standard Library (net/http) | 1.22+ | HTTP routing and middleware (no external framework) |
| gRPC | grpc-go | Latest | Internal microservice communication |
| Database Driver | pgx | v5 | PostgreSQL driver with connection pooling |
| Migration Tool | golang-migrate/migrate | v4 | Database schema migrations |
| Database | PostgreSQL | 15+ | Catalog database |
| Object Storage | Google Cloud Storage | N/A | Photo and thumbnail storage |
| Container Runtime | Docker | Latest | Containerization |
| Monitoring | Prometheus + Grafana | Latest | Metrics and dashboards |
| Logging | Structured (zerolog or zap) | Latest | JSON-formatted logs |

**Why Standard Library (net/http)?**
- Zero external dependencies for HTTP handling
- Full control over routing and middleware chain
- Lightweight and performant
- Easier to reason about request flow
- Standard library is battle-tested and well-documented
- Explicit middleware pattern (no magic)

**Handler Organization:**
```go
// Example structure using net/http
package main

import (
    "net/http"
)

func main() {
    mux := http.NewServeMux()
    
    // Routes
    mux.HandleFunc("POST /api/v1/photos/upload-url", handlers.GetSignedUploadURL)
    mux.HandleFunc("POST /api/v1/photos/{id}/new-signed-url", handlers.GetNewSignedURL)  // Retry endpoint
    mux.HandleFunc("POST /api/v1/photos/confirm", handlers.ConfirmUpload)
    mux.HandleFunc("GET /api/v1/photos/{id}", handlers.GetPhoto)
    mux.HandleFunc("GET /api/v1/photos", handlers.BrowsePhotos)
    
    // Middleware chain
    handler := middleware.Chain(
        mux,
        middleware.Logging,
        middleware.Recovery,
        middleware.APIKeyAuth,
        middleware.RateLimit,
        middleware.CORS,
    )
    
    http.ListenAndServe(":8080", handler)
}
```

### 5.2 Database Schema (Simplified)

```sql
-- API Keys table
CREATE TABLE api_keys (
    key_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    scope VARCHAR(50)[] NOT NULL, -- {'read', 'write', 'admin'}
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE
);

-- Pending uploads (for signed URL tracking)
CREATE TABLE pending_uploads (
    upload_token UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id UUID NOT NULL UNIQUE REFERENCES photos(photo_id),
    api_key_id UUID NOT NULL REFERENCES api_keys(key_id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'expired'))
);

CREATE INDEX idx_pending_uploads_token ON pending_uploads(upload_token);
CREATE INDEX idx_pending_uploads_api_key ON pending_uploads(api_key_id) WHERE status = 'pending';

-- Photos catalog
CREATE TABLE photos (
    photo_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_id VARCHAR(50) NOT NULL,
    lane_code VARCHAR(10) NOT NULL CHECK (lane_code ~ '^(L|R)[1-9]$|^(L|R)10$'),
    
    -- Coordinates (EPSG:4326)
    latitude DECIMAL(10, 8) NOT NULL CHECK (latitude >= -90 AND latitude <= 90),
    longitude DECIMAL(11, 8) NOT NULL CHECK (longitude >= -180 AND longitude <= 180),
    
    -- Linear Reference System data (optional - can be filled by other services)
    sta_value DECIMAL(10, 2),
    sta_source VARCHAR(20) CHECK (sta_source IN ('user_provided', 'lrs_interpolated')),
    
    -- Storage paths
    gcs_object_name VARCHAR(500) NOT NULL,
    
    -- File metadata
    file_format VARCHAR(10) NOT NULL CHECK (file_format IN ('JPEG', 'PNG')),
    file_size_bytes BIGINT NOT NULL,
    original_filename VARCHAR(255),
    
    -- User-provided metadata
    description TEXT,
    tags TEXT[],
    
    -- Upload tracking (new fields for retry support)
    upload_status VARCHAR(20) DEFAULT 'pending' CHECK (upload_status IN ('pending', 'completed', 'expired')),
    retry_count INTEGER DEFAULT 0 CHECK (retry_count >= 0 AND retry_count <= 5),
    
    -- Upload metadata
    uploaded_by_api_key UUID REFERENCES api_keys(key_id),
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Soft delete
    deleted_at TIMESTAMP WITH TIME ZONE,
    deleted_by_api_key UUID REFERENCES api_keys(key_id),
    
    -- Indexes created separately
);

-- Create indexes for performance
CREATE INDEX idx_photos_route_id ON photos(route_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_sta_value ON photos(route_id, sta_value) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_route_lane ON photos(route_id, lane_code, sta_value) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_coordinates ON photos(latitude, longitude) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_uploaded_at ON photos(uploaded_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_photos_upload_status ON photos(upload_status) WHERE deleted_at IS NULL;

-- Audit log for photo operations
CREATE TABLE photo_audit_log (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id UUID REFERENCES photos(photo_id),
    operation VARCHAR(20) NOT NULL, -- 'upload_signed_url', 'upload_confirm', 'update', 'delete'
    operated_by_api_key UUID REFERENCES api_keys(key_id),
    operated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    operation_details JSONB
);
```

**Schema Changes from Original Design:**
1. ~~Removed `status` column from photos~~ **Re-added**: `upload_status` column to track pending/completed/expired states
2. Removed `thumbnail_*_path` columns (thumbnails deferred to future phase)
3. Removed `exif_data` column (EXIF extraction deferred to future phase)
4. Simplified `pending_uploads` to store only token, photo_id, api_key_id
5. Made `sta_value` and `sta_source` nullable (filled by other services later)
6. Removed `processing_completed_at` column
7. **Added**: `retry_count` column to track upload retry attempts (max 5)
8. **Added**: `upload_status` ENUM ('pending', 'completed', 'expired') for upload lifecycle tracking

### 5.3 Service Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Web Application (React)                   │
│                 Running in User's Browser                    │
│                   (REST API Client Only)                     │
└─────────────────┬───────────────────────────┬───────────────┘
                  │                           │
                  │1. Request Signed URL      │3. Upload Photo
                  │   (REST API)               │   (Direct HTTP PUT)
                  ▼                           ▼
      ┌─────────────────────────────────────────────┐
      │      Bina Marga Survey Photo Service        │
      │              (Golang Application)             │
      │                                              │
      │  ┌──────────────┐      ┌──────────────┐    │
      │  │ REST Handler │      │ gRPC Handler │    │
      │  │  (net/http)  │      │(Browsing Only)│   │
      │  │ - Uploads     │      │ - GetPhoto    │    │
      │  │ - Downloads   │      │ - Browse      │    │
      │  │ - Browsing    │      │ - Search      │    │
      │  └──────┬───────┘      └──────┬───────┘    │
      │         │                     │            │
      │         │    ┌────────────────┘            │
      │         │    │ (Browsing queries)          │
      │         │    │ (Internal services)          │
      │         │    ▼                              │
      │         └──────────┐                       │
      │                    │        ┌──────────┐   │
      │         ┌──────────▼───────┐│ Internal │   │
      │         │   Service Layer   ││ Services │   │
      │         │  (Business Logic) │└──────────┘   │
      │         └──────────┬──────────┘            │
      │                    │                       │
      │         ┌──────────▼──────────┐            │
      │         │ Repository Layer     │            │
      │         │  (Data Access)       │            │
      │         └─────┬─────────┬──────┘            │
      │               │         │                   │
      │               │         │                   │
      │    ┌──────────▼───┐  ┌──▼─────────────┐    │
      │    │ PostgreSQL   │  │ Google Cloud    │    │
      │    │   (pgx)      │  │ Storage Client │    │
      │    └──────────────┘  └────────────────┘    │
      │                                              │
      │         ┌─────────────────────┐            │
      │         │ LRS Service Client  │            │
      │         │      (gRPC)          │            │
      │         └──────────┬──────────┘            │
      └────────────────────┼───────────────────────┘
                           │
                  ┌────────▼─────────┐
                  │  LRS Service      │
                  │  (External, gRPC) │
                  └──────────────────┘

Protocol Usage Summary:
──────────────────────────────────────────────────────────
REST API (net/http):
  - All client-facing operations (React web app)
  - Photo uploads (signed URLs + completion)
  - Photo downloads
  - Photo management (update, delete)
  - Health checks

gRPC API:
  - Internal microservice catalog browsing only
  - GetPhoto, BrowsePhotos, SearchPhotos, BatchGetPhotos
  - Called by other Bina Marga services needing photo data
──────────────────────────────────────────────────────────
```

### 5.4 Component Responsibilities

**1. REST Handler Layer (net/http)**
- HTTP request/response handling for all client operations
- Input validation and sanitization
- API Key authentication middleware
- Rate limiting middleware
- Request routing for uploads, downloads, and browsing
- JSON request/response serialization

**2. gRPC Handler Layer**
- gRPC service implementation for catalog browsing ONLY
- Protocol buffer serialization
- Internal service authentication (mTLS or API key)
- RPC method routing: GetPhoto, BrowsePhotos, SearchPhotos, BatchGetPhotos
- NO upload operations (uploads use REST API exclusively)

**3. Service Layer**
- Business logic orchestration
- Workflow coordination for uploads
- LRS integration coordination
- Thumbnail generation orchestration
- EXIF extraction coordination
- Error handling and logging
- Shared logic used by both REST and gRPC handlers

**4. Repository Layer**
- Database operations (CRUD)
- Connection pooling management
- Transaction management
- Query optimization

**5. External Clients**
- GCS Client: Photo and thumbnail storage (used by REST handlers)
- LRS Client: STA interpolation and validation (gRPC client to external service)

---

## 6. API Specification

### 6.0 API Protocol Usage

**REST API:** Used for all client-facing operations including:
- Photo upload (signed URL generation, upload completion)
- Photo management (update metadata, delete)
- Photo browsing and search
- Health checks and status endpoints

**gRPC API:** Used for internal microservice-to-microservice communication only:
- Photo catalog browsing by route, STA, lane
- Photo search with complex queries
- Batch photo metadata retrieval
- Real-time catalog queries from other services

**Why this separation?**
- Uploads require multipart/form-data or two-phase signed URLs (better suited for REST)
- Browsing/search benefits from gRPC's strongly-typed protocol buffers
- Internal services have lower latency requirements and benefit from gRPC
- External clients (React web app) use REST for simplicity

### 6.1 REST API Endpoints

#### 6.1.1 Authentication Header
All requests must include:
```json
X-API-Key: {api_key}
```

#### 6.1.2 Get Signed Upload URL (Phase 1)

**Note:** Phase 1 now accepts all photo attributes upfront. This allows full validation before generating the signed URL. Only Route ID and Lane Code are required.

```json
POST /api/v1/photos/upload-url
Content-Type: application/json

Request:
{
  "file_metadata": {
    "filename": "survey_photo_001.jpg",
    "content_type": "image/jpeg",
    "file_size_bytes": 2048576
  },
  "photo_attributes": {
    "route_id": "NR-001",
    "lane_code": "L1",
    "description": "Road surface damage at kilometer 5",
    "tags": ["damage", "road_surface"]
  }
}

Response (201 Created):
{
  "upload_token": "uuid-token-for-upload-tracking",
  "signed_url": "https://storage.googleapis.com/bm-survey-photos/...",
  "photo_id": "pre-generated-uuid",
  "expires_at": "ISO8601 timestamp (15 minutes from now)"
}
```

**Backend Actions:**
1. Validate API key
2. Validate file metadata (size, content type)
3. Validate photo attributes (route_id, lane_code, coordinates, sta_value)
4. Check concurrent upload limit (max 10 pending per API key)
5. Generate photo_id, upload_token, gcs_object_name
6. Create photo record in database
7. Create pending_upload record
8. Generate and return signed URL

**Notes:**
- Signed URL is valid for 15 minutes
- Client must upload directly to GCS using the signed URL
- Upload token must be saved for Phase 2 confirmation
- Photo metadata is stored in database during Phase 1
- **Retry**: If GCS upload fails, use `POST /photos/{photo_id}/new-signed-url` to get a new signed URL for the same photo

#### 6.1.3 Get New Signed URL for Retry

**Purpose:** Generate a new signed URL for an existing pending photo when the initial GCS upload fails.

```http
POST /api/v1/photos/{photo_id}/new-signed-url
Content-Type: application/json
X-API-Key: {api_key}
```

**Request:**
- No request body required (uses photo_id from URL path)

**Response (200 OK):**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "upload_token": "new-uuid-token-for-retry",
  "signed_url": "https://storage.googleapis.com/bm-survey-photos/...",
  "expires_at": "2024-01-15T10:45:00Z",
  "retry_count": 1,
  "max_retries": 5
}
```

**Backend Actions:**
1. Validate API key
2. Look up photo by photo_id
3. Verify photo exists and upload_status = 'pending'
4. Verify requesting API key matches the one that created the photo
5. Check retry count (max 5 attempts per photo)
6. Mark all existing pending_upload tokens for this photo as 'expired'
7. Generate new upload_token and signed_url (15 min expiry)
8. Create new pending_upload record
9. Increment retry_count on photo record
10. Return new credentials to client

**Error Responses:**
- 401 Unauthorized: Invalid API key
- 404 Not Found: Photo ID does not exist
- 409 Conflict: Photo already completed or deleted
- 403 Forbidden: Photo created by different API key
- 429 Too Many Requests: Retry limit (5) exceeded

**Important:** The GCS object name remains the same across retries. Only the signed URL and upload token change.

#### 6.1.5 Upload to Google Cloud Storage (Client-Side)
```
PUT {signed_url}
Content-Type: image/jpeg

Request:
- Binary file data

Response (200 OK from GCS):
ETag: "etag-hash"
```

**Client-Side Implementation Notes:**
- Use XMLHttpRequest or fetch API with PUT method
- Set Content-Type header to match content_type from Phase 1
- Track upload progress using progress events
- Do NOT include authentication headers (signed URL has auth embedded)
- Handle CORS configuration in GCS bucket settings

#### 6.1.6 Confirm Upload (Phase 2)

**Note:** Phase 2 is a simple confirmation that the client successfully uploaded the file to GCS. All photo metadata was already provided in Phase 1.

```
POST /api/v1/photos/confirm
Content-Type: application/json

Request:
{
  "upload_token": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}

Response (200 OK):
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Upload confirmed successfully"
}
```

**Backend Actions:**
1. Validate API key
2. Validate upload_token (exists, not expired, status='pending')
3. Verify API key matches token record
4. Verify file exists in GCS at gcs_object_name
5. Mark pending_upload status as 'completed'

**Error Responses:**
- 400 Bad Request: Invalid or missing upload_token
- 404 Not Found: Token not found
- 409 Conflict: Token already used or expired
- 404 Not Found: File not found in GCS

### 6.1.7 Benefits of Two-Phase Upload Architecture

**Performance Benefits:**
- Reduced server load: Files upload directly to GCS, bypassing backend
- Faster uploads: Direct GCS uploads leverage Google's infrastructure
- Better scalability: Backend handles metadata only, not file streaming
- Progress tracking: Client can show real-time upload progress to users

**Architecture Benefits:**
- Clear separation of concerns: Upload vs metadata processing
- Resumable uploads: Easier to implement retry logic for failed uploads
- Queue-based processing: Thumbnails can be generated asynchronously
- Parallel processing: Multiple photos can upload simultaneously without blocking

**Client Experience:**
- Real-time upload progress bars
- Better error handling with clear feedback
- Ability to cancel uploads in progress
- Resume interrupted uploads without restarting

#### 6.1.8 Browse Photos

```http
GET /api/v1/photos?route_id={route_id}&sta_start={start}&sta_end={end}&lane={lane}&page={page}&per_page={per_page}
```

**Response (200 OK):**
```json
{
  "photos": [
    {
      "photo_id": "550e8400-e29b-41d4-a716-446655440000",
      "route_id": "NR-001",
      "lane_code": "L1",
      "sta_value": 5.2,
      "thumbnail_url": "https://storage.googleapis.com/.../small.jpg",
      "uploaded_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 20,
    "total_count": 100,
    "total_pages": 5
  }
}
```

#### 6.1.9 Get Photo Metadata

```http
GET /api/v1/photos/{photo_id}
```

**Response (200 OK):**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "route_id": "NR-001",
  "lane_code": "L1",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "sta_value": 5.2,
  "sta_source": "user_provided",
  "file_format": "JPEG",
  "file_size_bytes": 2048576,
  "exif_data": {
    "timestamp": "2024-01-15T10:25:00Z",
    "camera_make": "Canon",
    "camera_model": "EOS R5",
    "gps_latitude": -6.2088,
    "gps_longitude": 106.8456,
    "altitude": 15.5,
    "orientation": 1
  },
  "description": "Road surface damage at kilometer 5",
  "tags": ["damage", "road_surface"],
  "uploaded_at": "2024-01-15T10:30:00Z",
  "download_url": "https://storage.googleapis.com/...",
  "thumbnail_urls": {
    "small": "https://storage.googleapis.com/.../small.jpg",
    "medium": "https://storage.googleapis.com/.../medium.jpg",
    "large": "https://storage.googleapis.com/.../large.jpg"
  }
}
```

#### 6.1.10 Download Photo

```http
GET /api/v1/photos/{photo_id}/download
```

**Response (200 OK):**
```
Content-Type: image/jpeg
Content-Disposition: attachment; filename="{photo_id}.jpg"

(Binary file stream)
```

#### 6.1.11 Download Thumbnail

```http
GET /api/v1/photos/{photo_id}/thumbnail?size={small|medium|large}
```

**Response (200 OK):**
```
Content-Type: image/jpeg

(Binary thumbnail file stream)
```

#### 6.1.12 Update Photo Metadata

```http
PATCH /api/v1/photos/{photo_id}
Content-Type: application/json
```

**Editable Fields:**
- `description` (string, optional): Photo description
- `tags` (array of strings, optional): Tags for categorization
- `lane_code` (string, optional): Lane identifier (L1-L10 or R1-R10)
- `latitude` (decimal, optional): Latitude coordinate (EPSG:4326)
- `longitude` (decimal, optional): Longitude coordinate (EPSG:4326)
- `sta_value` (decimal, optional): Station value along route

**Validation Rules:**
- Both `latitude` and `longitude` must be provided together (partial updates rejected)
- Latitude must be between -90 and 90
- Longitude must be between -180 and 180
- STA value must be >= 0
- When `sta_value` is updated, `sta_source` is set to `user_provided`

**Request:**
```json
{
  "description": "Updated description",
  "tags": ["damage", "road_surface", "urgent"],
  "lane_code": "R2",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "sta_value": 1234.5
}
```

**Response (200 OK):**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Updated description",
  "tags": ["damage", "road_surface", "urgent"],
  "lane_code": "R2",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "sta_value": 1234.5,
  "sta_source": "user_provided",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

**Error Responses:**
- 400 Bad Request: Invalid field values or partial coordinate update
- 404 Not Found: Photo ID does not exist
- 401 Unauthorized: Invalid API key

#### 6.1.13 Delete Photo

```http
DELETE /api/v1/photos/{photo_id}?hard={true|false}
```

**Response (200 OK):**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "deleted_at": "2024-01-15T12:00:00Z",
  "deletion_type": "soft"
}
```

**Query Parameters:**
- `hard=true`: Permanently delete from database and GCS
- `hard=false` (default): Soft delete (mark as deleted, retain files)

#### 6.1.14 Health Check

```http
GET /health
```

**Response (200 OK):**
```json
{
  "status": "healthy",
  "database": "connected",
  "storage": "connected",
  "lrs_service": "connected",
  "version": "1.0.0"
}
```

### 6.2 gRPC Service Definition (Draft)

**Purpose:** gRPC is used ONLY for catalog browsing by internal microservices. Photo uploads use REST API exclusively.

**To Be Provided:** Await LRS service team's gRPC specification format to ensure consistency.

**Proposed Service:**
```protobuf
syntax = "proto3";

package bina_marga.survey_photo.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";

// Service for browsing photo catalog (internal microservices only)
// Upload operations use REST API
service PhotoCatalogService {
  // Get a single photo by ID
  rpc GetPhoto(GetPhotoRequest) returns (GetPhotoResponse);
  
  // Browse photos by route with optional STA and lane filters
  rpc BrowsePhotos(BrowsePhotosRequest) returns (BrowsePhotosResponse);
  
  // Advanced search with multiple filters
  rpc SearchPhotos(SearchPhotosRequest) returns (SearchPhotosResponse);
  
  // Get photo status (processing, ready, failed)
  rpc GetPhotoStatus(GetPhotoStatusRequest) returns (GetPhotoStatusResponse);
  
  // Batch get multiple photos by IDs
  rpc BatchGetPhotos(BatchGetPhotosRequest) returns (BatchGetPhotosResponse);
}

// Browse photos by route, STA range, and lane
message BrowsePhotosRequest {
  string route_id = 1;
  optional double sta_start = 2;
  optional double sta_end = 3;
  string lane_code = 4;
  int32 page = 5;
  int32 per_page = 6;
}

message BrowsePhotosResponse {
  repeated PhotoMetadata photos = 1;
  Pagination pagination = 2;
}

message Pagination {
  int32 current_page = 1;
  int32 per_page = 2;
  int64 total_count = 3;
  int32 total_pages = 4;
}

// Get single photo
message GetPhotoRequest {
  string photo_id = 1;
}

message GetPhotoResponse {
  PhotoMetadata photo = 1;
  string download_url = 2;
  map<string, string> thumbnail_urls = 3; // small, medium, large
}

// Advanced search
message SearchPhotosRequest {
  repeated string route_ids = 1;
  repeated double sta_ranges = 2; // pairs of [start, end]
  repeated int32 lanes = 3;
  optional google.protobuf.Timestamp date_start = 4;
  optional google.protobuf.Timestamp date_end = 5;
  repeated string tags = 6;
  int32 page = 7;
  int32 per_page = 8;
}

message SearchPhotosResponse {
  repeated PhotoMetadata photos = 1;
  Pagination pagination = 2;
}

// Photo status
message GetPhotoStatusRequest {
  string photo_id = 1;
}

message GetPhotoStatusResponse {
  string photo_id = 1;
  string status = 2; // processing, ready, failed
  bool thumbnails_ready = 3;
  bool exif_extracted = 4;
  bool sta_calculated = 5;
  optional string error_message = 6;
  optional google.protobuf.Timestamp processed_at = 7;
}

// Batch get
message BatchGetPhotosRequest {
  repeated string photo_ids = 1;
}

message BatchGetPhotosResponse {
  repeated PhotoMetadata photos = 1;
}

// Common message types
message PhotoMetadata {
    string photo_id = 1;
    string route_id = 2;
    string lane_code = 3;
    double latitude = 4;
    double longitude = 5;
    double sta_value = 6;
    string sta_source = 7;
    string file_format = 8;
    int64 file_size_bytes = 9;
    string description = 10;
    repeated string tags = 11;
    google.protobuf.Timestamp uploaded_at = 12;
    string status = 13;
}
```

**gRPC Method Mapping:**

| gRPC Method | REST Equivalent | Purpose |
|------------|----------------|---------|
| GetPhoto | GET /api/v1/photos/{id} | Get single photo |
| BrowsePhotos | GET /api/v1/photos | Browse by route/STA/lane |
| SearchPhotos | POST /api/v1/photos/search | Advanced search |
| GetPhotoStatus | GET /api/v1/photos/{id}/status | Check status |
| BatchGetPhotos | N/A (internal only) | Batch retrieval for internal services |

**Note:** Photo upload operations (GetSignedUploadURL, CompleteUpload) are REST-only and NOT exposed via gRPC.

---

## 7. Development Phases

### Phase 1: Foundation (Weeks 1-2) - ✅ COMPLETE (Domain Layer)

**Deliverables:**
- ~~Project structure and boilerplate~~ ✅
- ~~Database schema design with golang-migrate migration files~~ ⚠️ Planned
- ~~Basic pgx connection pooling~~ ⚠️ Planned
- ~~golang-migrate integration for schema versioning~~ ⚠️ Planned
- ~~Configuration management (environment variables)~~ ⚠️ Planned
- ~~Structured logging setup~~ ⚠️ Planned
- ~~Health check endpoint~~ ⚠️ Planned
- ✅ **Domain models (VOs, entities, DTOs)**
- ✅ **Domain errors and constants**

**Acceptance Criteria:**
- [x] Database migrations run successfully with `migrate up` - Pending
- [x] Health check endpoint returns ok status - Pending
- [x] Application starts and connects to database - Pending
- [x] Migration rollback works correctly (`migrate down`) - Pending
- [x] Domain layer implemented with tests - **COMPLETE**

### Phase 2: Core Upload Functionality (Weeks 3-4)

**Deliverables:**
- REST API handlers for upload workflow (signed URL + completion)
- File validation (format, size)
- Google Cloud Storage integration with signed URLs
- Upload token tracking and state management
- EXIF metadata extraction
- Thumbnail generation (background process)
- Basic error handling

**Acceptance Criteria:**
- Clients can request signed URLs via REST API
- Photos upload to GCS successfully using signed URLs
- Completion endpoint validates tokens correctly
- Thumbnails generate correctly
- EXIF data extracts and stores properly
- Errors return appropriate HTTP status codes

### Phase 3: LRS Integration (Week 5)

**Deliverables:**
- gRPC client for LRS service
- STA interpolation logic
- STA validation logic
- Retry and circuit breaker pattern
- Fallback handling when LRS unavailable

**Acceptance Criteria:**
- LRS interpolation works for valid coordinates
- Validation works for user-provided STA values
- Graceful degradation when LRS is unavailable
- Errors logged appropriately

### Phase 4: Browse and Search (Week 6)

**Deliverables:**
- REST API browse endpoints (by route, STA, lane)
- gRPC service for catalog browsing (GetPhoto, BrowsePhotos, SearchPhotos)
- gRPC service for internal microservices (BatchGetPhotos)
- Pagination implementation (REST and gRPC)
- Download endpoints (original and thumbnails) - REST only
- Query optimization (indexes)
- Advanced search (if time permits)

**Acceptance Criteria:**
- REST browse returns paginated results
- gRPC BrowsePhotos works for internal services
- Queries perform within 100ms p95
- Downloads work correctly via REST
- Results filter by route, STA, lane correctly

### Phase 5: Management & Admin (Week 7)

**Deliverables:**
- API Key management endpoints
- Update photo metadata endpoint
- Delete photo endpoint (soft/hard delete)
- Bulk delete (if time permits)
- Audit logging

**Acceptance Criteria:**
- API Key creation and revocation works
- Metadata updates persist correctly
- Soft and hard delete work as specified
- Audit logs capture all operations

### Phase 6: Testing & Hardening (Week 8)

**Deliverables:**
- Unit tests for critical paths
- Integration tests for database operations
- Performance testing
- Security validation (input sanitization, SQL injection prevention)
- Load testing
- Documentation (API docs)

**Acceptance Criteria:**
- Core functionality has >80% code coverage
- Performance meets requirements (p95 <500ms)
- Security scan passes
- API documentation complete

---

## 8. Testing Strategy

### 8.1 Unit Testing
- **Framework:** Go testing package
- **Scope:** Service layer logic, validation functions, utilities
- **Target Coverage:** >80% for business logic
- **Mocking:** Use mocks for external dependencies (GCS, LRS, database)

### 8.2 Integration Testing
- **Scope:** Repository layer, API endpoints
- **Test Database:** Use testcontainers or Docker PostgreSQL
- **Cleanup:** Reset database state between tests

### 8.3 Performance Testing
- **Tool:** k6 or Vegeta
- **Scenarios:**
  - Upload throughput
  - Browse query latency
  - Concurrent user simulation
- **Metrics:**
  - Response times (p50, p95, p99)
  - Error rates
  - Database connection pool usage

### 8.4 Security Testing
- **SQL Injection:** Validate all inputs
- **File Upload:** Test malicious file uploads
- **API Key:** Test authentication and authorization
- **Rate Limiting:** Validate rate limits work

---

## 9. Deployment and DevOps

### 9.1 Infrastructure

**Compute Engine VM Specifications (MVP):**
- **Machine Type:** e2-standard-4 (4 vCPU, 16 GB memory)
- **OS:** Ubuntu 22.04 LTS
- **Storage:** 100 GB SSD persistent disk
- **Network:** VPC with private subnet
- **Region:** Indonesia (if available) or Singapore (asia-southeast1)

**Database (Cloud SQL):**
- **Instance:** PostgreSQL 15
- **Machine Type:** db-custom-2-7680 (2 vCPU, 7.5 GB memory)
- **Storage:** 100 GB SSD
- **High Availability:** Single instance (MVP), HA for production
- **Backup:** Daily automated backups, 30-day retention

**Google Cloud Storage:**
- **Bucket:** bm-survey-photos-dev/prod
- **Location:** Regional (Indonesia or Singapore)
- **Storage Class:** Standard
- **Versioning:** Disabled (MVP), enable for production

### 9.2 Deployment Architecture

```
┌─────────────────────────────────────────────────────┐
│                 Google Cloud Platform                │
│                                                      │
│  ┌──────────────┐        ┌─────────────────────┐    │
│  │ Cloud SQL     │        │ Compute Engine VM   │    │
│  │ PostgreSQL   │◄───────┤ Bina Marga Photo    │    │
│  │              │        │ Service             │    │
│  └──────────────┘        │                     │    │
│                          │ - Golang App         │    │
│                          │ - Docker Container   │    │
│                          └─────────┬───────────┘    │
│                                    │                 │
│                          ┌─────────▼───────────┐    │
│                          │ Google Cloud Storage│    │
│                          │ - Original Photos   │    │
│                          │ - Thumbnails         │    │
│                          └─────────────────────┘    │
│                                                      │
│  ┌──────────────────────────────────────────┐       │
│  │ External: Linear Reference System Service │       │
│  │ (gRPC connection via VPC peering)         │       │
│  └──────────────────────────────────────────┘       │
│                                                      │
│  Monitoring:                                         │
│  ┌──────────────┐        ┌─────────────────────┐   │
│  │ Prometheus    │        │ Grafana Dashboard    │   │
│  │ (on VM)       │───────►│ - API Metrics        │   │
│                │        │ - Database Metrics    │   │
│                │        │ - Custom Dashboards   │   │
│                └────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### 9.3 Configuration Management
- **Environment Variables:**
  - DATABASE_URL
  - GCS_BUCKET_NAME
  - LRS_SERVICE_HOST
  - LRS_SERVICE_PORT
  - API_KEY_ENCRYPTION_KEY
  - LOG_LEVEL
  - PORT

- **Secrets Management:**
  - Use Google Secret Manager for sensitive configuration
  - Never commit secrets to version control

### 9.4 Backup and Recovery
- **Database:** Cloud SQL automated daily backups
- **Photos:** GCS multi-regional redundancy (if needed)
- **Application:** Infrastructure as Code (Terraform) for reprovisioning

---

## 10. Monitoring and Observability

### 10.1 Prometheus Metrics

**Application Metrics:**
- `photo_upload_total{status}` - Count of photo uploads by status
- `photo_download_total{status}` - Count of photo downloads
- `api_request_duration_seconds{method, endpoint}` - Request latency
- `api_request_total{method, endpoint, status}` - Request count
- `active_api_keys_count` - Number of active API keys

**Database Metrics:**
- `db_query_duration_seconds{query_type}` - Query latency
- `db_connection_pool_active` - Active connections
- `db_connection_pool_idle` - Idle connections
- `db_connection_pool_wait_count` - Waiting connections

**Storage Metrics:**
- `gcs_upload_duration_seconds` - GCS upload latency
- `gcs_download_duration_seconds` - GCS download latency
- `gcs_operation_total{operation, status}` - GCS operation count

**LRS Service Metrics:**
- `lrs_request_duration_seconds{operation}` - LRS call latency
- `lrs_request_total{operation, status}` - LRS request count

### 10.2 Logging Standards

**Log Format (JSON):**
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "message": "Photo uploaded successfully",
  "request_id": "uuid",
  "api_key_id": "uuid",
  "photo_id": "uuid",
  "route_id": "string",
  "sta_value": decimal,
  "duration_ms": integer,
  "metadata": {}
}
```

**Log Levels:**
- ERROR: Application errors, external service failures
- WARN: Deprecated usage, potential issues
- INFO: Normal operations (uploads, downloads, queries)
- DEBUG: Detailed debug information (disabled in production)

### 10.3 Grafana Dashboards

**Dashboard 1: API Overview**
- Request rate by endpoint
- Response time percentiles (p50, p95, p99)
- Error rate by endpoint
- Active requests

**Dashboard 2: Upload Pipeline**
- Upload success/failure rate
- Upload duration histogram
- EXIF extraction duration
- Thumbnail generation duration
- LRS interpolation success rate
- GCS upload duration

**Dashboard 3: Database Health**
- Connection pool usage
- Query duration by type
- Transaction duration
- Lock wait time

**Dashboard 4: Storage Metrics**
- GCS operation count
- GCS operation latency
- Storage usage by bucket
- Signed URL generation count

---

## 11. Security Considerations

### 11.1 API Key Management
- **Generation:** Cryptographically secure random UUIDs
- **Storage:** Hash API keys using bcrypt before storing
- **Rotation:** Support key rotation with grace period
- **Revocation:** Immediate revocation capability
- **Scope Validation:** Check scopes on every request

### 11.2 Signed URL Security

**Single-Use Enforcement:**
- Signed URLs are designed for single upload attempt only
- Upload tokens track state transitions: pending → uploaded → completed
- Backend validates token state before processing metadata
- Prevents:
  - File overwrites by malicious actors
  - Unauthorized uploads using stolen URLs
  - Accidental duplicate uploads

**Expiration and Lifecycle:**
- Signed URLs expire after 15 minutes
- Upload tokens expire after 15 minutes if not used
- Expired tokens are cleaned up automatically (background job runs hourly)
- Maximum concurrent pending uploads per API key (default: 5) to prevent abuse

**Signed URL Generation Process:**
```
1. Validate API key and permissions
2. Validate file metadata (size, type)
3. Generate unique photo_id and GCS object name
4. Create upload_token (UUID)
5. Store pending_upload record with status='pending' ←
6. Generate signed URL using GCS service account
   - Method: PUT
   - Expires: 15 minutes from now
   - Content-Type: image/jpeg or image/png
   - Content-Length: file_size_bytes
7. Return signed URL and upload_token to client
```

**Upload Completion Validation:**
```
1. Verify upload_token exists in database
2. Verify upload_token.status == 'uploaded' (NOT pending, completed, or expired)
3. Verify upload_token not expired (created_at + 15 minutes > now)
4. Verify API key matches token record
5. Retrieve gcs_object_name from photo record (client does NOT send this)
6. Mark token as 'processing' (prevents concurrent completions)
7. Verify file exists in GCS at gcs_object_name
8. Process metadata and generate thumbnails
9. Mark token as 'completed'
10. Return success response
```

**Concurrent Request Protection:**
- If two completion requests arrive simultaneously for same token
- One request will succeed in marking token as 'processing'
- Other request will receive UPLOAD_IN_PROGRESS error
- Client should retry after checking photo status

**GCS CORS Configuration:**
```
[
  {
    "origin": ["https://app.bina-marga.go.id"],
    "method": ["PUT"],
    "responseHeader": ["Content-Type", "Content-Length"],
    "maxAgeSeconds": 3600
  }
]
```

### 11.3 Rate Limiting for Signed URLs
- **Get Signed URL:** 10 requests per minute per API key
- **Complete Upload:** 10 requests per minute per API key
- **Concurrent Pending Uploads:** Maximum 5 per API key
- **Rationale:**
  - Prevents signed URL abuse
  - Limits storage of pending_upload records
  - Prevents resource exhaustion

### 11.3 Input Validation
- **File Metadata:**
  - Validate filename for malicious characters
  - Validate content_type matches allowed types
  - Validate file_size is within limits (10MB)
- **Photo Metadata:**
  - Validate latitude/longitude ranges
  - Validate lane number is positive integer
  - Sanitize text inputs (description, tags)
  - Validate route_id format

### 11.4 SQL Injection Prevention
- Use parameterized queries exclusively
- Never construct SQL from user input
- Validate input types before database operations

### 11.5 Rate Limiting
- **Implementation:** Token bucket algorithm
- **Limits per API Key:**
  - Signed URL requests: 10 per minute
  - Complete upload requests: 10 per minute
  - Browse requests: 100 per minute
  - Total requests: 100 per minute
- **Response:** HTTP 429 Too Many Requests with Retry-After header
- **Concurrent Uploads:** Limit pending uploads per API key (default: 5)

### 11.6 HTTPS and TLS
- **Requirement:** TLS 1.3 for all API communication
- **Certificate:** Managed SSL certificate via Google Cloud
- **In-Transit Encryption:** All data encrypted in transit
- **GCS Signed URLs:** HTTPS-only for signed URLs

### 11.7 Data Privacy
- **EXIF Data:** May contain personal information (GPS, timestamp, device)
- **Retention:** Consider EXIF data retention policies
- **Anonymization:** Option to strip sensitive EXIF fields
- **Original Filenames:** Store original filename separately, use UUID in GCS

### 11.8 Client-Side Security
- **CORS Configuration:** Configure GCS bucket CORS settings
  - Allowed Origins: Whitelist specific domains in production
  - Allowed Methods: PUT only for upload endpoint
  - Allowed Headers: Content-Type, Content-Length
  - Max Age: 3600 seconds
- **No Credentials:** Do not send API keys with direct GCS upload
- **Token Handling:** Store upload_token securely in client state

---

## 12. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| LRS Service Unavailability | Medium | High | Circuit breaker pattern, graceful degradation, queue STA calculation for retry |
| Database Performance Issues | Medium | High | Connection pooling, query optimization, proper indexing, caching |
| GCS Storage Cost Overrun | Low | Medium | Set storage limits, implement auto-archiving policy, monitor usage |
| API Key Leakage | Medium | High | Encrypted storage, secure transmission, rotation support, monitoring |
| Large Upload Volume Spike | Low | Medium | Rate limiting, background thumbnail processing, queue-based processing |
| Coordinate/STA Mismatch | Medium | Medium | Validation via LRS service, confidence scoring, manual override capability |
| EXIF Privacy Issues | Low | High | Strip personal EXIF data, obtain user consent, document data retention |

---

## 13. Future Enhancements

**Phase 2 Features:**
- Web-based admin interface for photo management
- Mobile SDK for photo upload from field devices
- Batch upload API for bulk operations
- Machine learning-based photo classification
- Route condition analysis from photos

**Phase 3 Features:**
- Real-time photo streaming
- Advanced geospatial queries (PostGIS integration)
- Integration with GIS mapping tools
- Photo annotation and marking features
- Automated report generation

---

## 14. Open Questions

1. **LRS Service Specification:** What is the exact gRPC API specification for the existing LRS microservice? Need .proto file and endpoint details.

2. **Route Data Source:** Where does the master list of national routes come from? Should we maintain a local cache or query LRS service?

3. **Authentication Authority:** Who is responsible for issuing and managing API keys? Is there a central auth service?

4. **Photo Deletion Policy:** What is the approval workflow for manual deletion? Who can authorize deletions?

5. **EXIF Privacy:** Should we automatically strip GPS coordinates from EXIF data to protect privacy?

6. **Backup/Disaster Recovery:** What is the Recovery Time Objective (RTO) and Recovery Point Objective (RPO) for this service?

7. **Multi-Region Deployment:** Is there a requirement for multi-region deployment for disaster recovery?

8. **SLA Requirements:** What are the service level agreement requirements for availability and response time?

---

## 15. Glossary

- **Bina Marga:** Indonesian Directorate General of Highways
- **STA:** Station value along a route (Linear Referencing System measure)
- **LRS:** Linear Referencing System - method to locate features along a linear route
- **EPSG:4326:** Coordinate reference system (WGS84 latitude/longitude)
- **EXIF:** Exchangeable Image File Format - metadata embedded in photo files
- **pgx:** PostgreSQL driver for Go with connection pooling
- **GCS:** Google Cloud Storage
- **UUID:** Universally Unique Identifier

---

## 16. Appendix

### Appendix A: Example API Request/Response (Two-Phase Upload)

**Phase 1: Get Signed Upload URL**

**Request:**
```bash
curl -X POST https://api.bina-marga.survey/v1/photos/upload-url \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "file_metadata": {
      "filename": "survey_photo_001.jpg",
      "content_type": "image/jpeg",
      "file_size_bytes": 2048576
    }
  }'
```

**Response:**
```json
{
  "upload_token": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "signed_url": "https://storage.googleapis.com/bm-survey-photos-dev/photos/2024/NR-001/NR-001_2024_L1_a1b2c3d4.jpg?Expires=1705312800&Signature=...",
  "gcs_object_name": "photos/2024/NR-001/NR-001_2024_L1_a1b2c3d4.jpg",
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2024-01-15T10:45:00Z"
}
```

**Note:** The `gcs_object_name` is returned for reference/debugging, but the client does NOT need to send it back in the complete request. The server uses `upload_token` to look up the file location.

**Phase 2: Upload to GCS (Client-Side)**

**Using curl (for testing):**
```bash
curl -X PUT "https://storage.googleapis.com/bm-survey-photos-dev/photos/2024/NR-001/NR-001_2024_L1_a1b2c3d4.jpg?Expires=1705312800&Signature=..." \
  -H "Content-Type: image/jpeg" \
  --data-binary "@/path/to/survey_photo_001.jpg"
```

**Phase 3: Complete Upload**

**Request:**
```bash
curl -X POST https://api.bina-marga.survey/v1/photos/complete \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "upload_token": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "route_id": "NR-001",
    "lane_code": "L1",
    "latitude": -6.2088,
    "longitude": 106.8456,
    "sta_value": 5.2,
    "description": "Road surface damage at kilometer 5",
    "tags": ["damage", "road_surface"],
    "upload_timestamp": "2024-01-15T10:30:00Z"
  }'
```

**Note:** The client does NOT send `gcs_object_name`. The server retrieves it from the `photos` table using the `photo_id` associated with the `upload_token`.

**Response:**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "route_id": "NR-001",
  "lane_code": "L1",
  "latitude": -6.2088,
  "longitude": 106.8456,
  "sta_value": 5.2,
  "sta_source": "user_provided",
  "file_format": "JPEG",
  "file_size_bytes": 2048576,
  "status": "processing",
  "uploaded_at": "2024-01-15T10:30:00Z",
  "thumbnail_urls": {
    "small": null,
    "medium": null,
    "large": null
  },
  "message": "Photo uploaded successfully. Thumbnails are being generated."
}
```

### Appendix B: React Client Example

**Photo Upload Component (TypeScript):**

```typescript
import React, { useState, useCallback } from 'react';

interface UploadProgress {
  phase: 'idle' | 'getting-url' | 'uploading' | 'completing' | 'done' | 'error';
  progress: number; //0-100
  message: string;
}

const API_BASE_URL = 'https://api.bina-marga.survey/v1';

export const PhotoUpload: React.FC = () => {
  const [file, setFile] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState<UploadProgress>({
    phase: 'idle',
    progress: 0,
    message: ''
  });
  const [metadata, setMetadata] = useState({
    routeId: '',
    laneNumber: 1,
    latitude: 0,
    longitude: 0,
    staValue: null as number | null,
    description: '',
    tags: [] as string[]
  });

  // Phase 1: Get signed upload URL from backend
  const getSignedUrl = async (file: File): Promise<{
    uploadToken: string;
    signedUrl: string;
    photoId: string;
  }> => {
    const response = await fetch(`${API_BASE_URL}/photos/upload-url`, {
      method: 'POST',
      headers: {
        'X-API-Key': process.env.REACT_APP_API_KEY!,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        file_metadata: {
          filename: file.name,
          content_type: file.type,
          file_size_bytes: file.size
        }
      })
    });

    if (!response.ok) {
      throw new Error('Failed to get signed URL');
    }

    const data = await response.json();
    return {
      uploadToken: data.upload_token,
      signedUrl: data.signed_url,
      photoId: data.photo_id
      // Note: gcs_object_name is returned but not needed by client
      // The server tracks it via photo_id linked to upload_token
    };
  };

  // Phase 2: Upload file directly to GCS using signed URL
  const uploadToGCS = async (file: File, signedUrl: string): Promise<void> => {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();

      // Track upload progress
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable) {
          const progress = Math.round((event.loaded / event.total) * 100);
          setUploadProgress(prev => ({
            ...prev,
            progress,
            message: `Uploading: ${progress}%`
          }));
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          setUploadProgress(prev => ({ ...prev, progress: 100 }));
          resolve();
        } else {
          reject(new Error(`Upload failed: ${xhr.status}`));
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });

      xhr.open('PUT', signedUrl);
      xhr.setRequestHeader('Content-Type', file.type);
      xhr.send(file);
    });
  };

  // Phase 3: Complete upload with metadata
  // Note: Only upload_token is needed - server looks up gcs_object_name
  const completeUpload = async (uploadToken: string): Promise<any> => {
    const response = await fetch(`${API_BASE_URL}/photos/complete`, {
      method: 'POST',
      headers: {
        'X-API-Key': process.env.REACT_APP_API_KEY!,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        upload_token: uploadToken,
        route_id: metadata.routeId,
        lane_code: metadata.laneCode,
        latitude: metadata.latitude,
        longitude: metadata.longitude,
        sta_value: metadata.staValue,
        description: metadata.description,
        tags: metadata.tags,
        upload_timestamp: new Date().toISOString()
      })
    });

    if (!response.ok) {
      throw new Error('Failed to complete upload');
    }

    return response.json();
  };

  const handleUpload = useCallback(async () => {
    if (!file) return;

    try {
      // Phase 1: Get signed URL
      setUploadProgress({ phase: 'getting-url', progress: 0, message: 'Preparing upload...' });
      const { uploadToken, signedUrl, photoId } = await getSignedUrl(file);

      // Phase 2: Upload to GCS
      setUploadProgress({ phase: 'uploading', progress: 0, message: 'Uploading photo...' });
      await uploadToGCS(file, signedUrl);

      // Phase 3: Complete upload (server uses upload_token to find file location)
      setUploadProgress({ phase: 'completing', progress: 100, message: 'Processing metadata...' });
      const result = await completeUpload(uploadToken);

      setUploadProgress({
        phase: 'done',
        progress: 100,
        message: `Upload complete! Photo ID: ${result.photo_id}`
      });

      console.log('Upload complete:', result);
    } catch (error) {
      setUploadProgress({
        phase: 'error',
        progress: 0,
        message: `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
      });
    }
  }, [file, metadata]);

  return (
    <div>
      <input type="file" accept="image/jpeg,image/png" onChange={(e) => {
        const selectedFile = e.target.files?.[0];
        if (selectedFile) setFile(selectedFile);
      }} />
      
      {/* Metadata inputs */}
      <input
        type="text"
        placeholder="Route ID"
        value={metadata.routeId}
        onChange={(e) => setMetadata({ ...metadata, routeId: e.target.value })}
      />
      
      {/* ... more metadata inputs ... */}
      
      <button onClick={handleUpload} disabled={!file || uploadProgress.phase !== 'idle'}>
        Upload Photo
      </button>
      
      {/* Progress display */}
      {uploadProgress.phase !== 'idle' && (
        <div>
          <p>{uploadProgress.message}</p>
          {uploadProgress.phase === 'uploading' && (
            <progress value={uploadProgress.progress} max="100" />
          )}
        </div>
      )}
    </div>
  );
};
```
      
      {/* Metadata inputs */}
      <input
        type="text"
        placeholder="Route ID"
        value={metadata.routeId}
        onChange={(e) => setMetadata({ ...metadata, routeId: e.target.value })}
      />
      
      {/* ... more metadata inputs ... */}
      
      <button onClick={handleUpload} disabled={!file || uploadProgress.phase !== 'idle'}>
        Upload Photo
      </button>
      
      {/* Progress display */}
      {uploadProgress.phase !== 'idle' && (
        <div>
          <p>{uploadProgress.message}</p>
          {uploadProgress.phase === 'uploading' && (
            <progress value={uploadProgress.progress} max="100" />
          )}
        </div>
      )}
    </div>
  );
};
```
  }
}
```

### Appendix B: Database Migration Strategy

**Migration Tool:** [golang-migrate/migrate](https://github.com/golang-migrate/migrate)

**Installation:**
```bash
# Install CLI tool
curl -L https://github.com/golang-migrate/migrate/releases/download/$version/migrate.$os-$arch.tar.gz | tar xvz
# Or use Go install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**Migration File Structure:**
```
migrations/
├── 000001_init_schema.up.sql
├── 000001_init_schema.down.sql
├── 000002_add_pending_uploads.up.sql
├── 000002_add_pending_uploads.down.sql
└── ...
```

**Commands:**
```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_table

# Apply migrations
migrate -database "postgres://user:pass@localhost:5432/bm_photos" -path migrations up

# Rollback last migration
migrate -database "postgres://user:pass@localhost:5432/bm_photos" -path migrations down 1

# Check migration version
migrate -database "postgres://user:pass@localhost:5432/bm_photos" -path migrations version
```

**Programmatic Usage (in Go):**
```go
package main

import (
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/file"
    _ "github.com/lib/pq"
)

func runMigrations(dbURL string) error {
    m, err := migrate.New(
        "file://migrations",
        dbURL)
    if err != nil {
        return err
    }
    defer m.Close()
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    return nil
}
```

**Versioning:** Sequential migration files with up/down migrations  
**Backups:** Always backup before migration  
**Testing:** Test migrations on staging environment first
**CI/CD:** Run migrations as part of deployment pipeline before starting the application  

### Appendix C: Error Response Format

**Standard Error Response:**
```json
{
  "error": {
    "code": "INVALID_COORDINATES",
    "message": "Latitude must be between -90 and 90 degrees",
    "details": {
      "field": "latitude",
      "value": -95.5
    }
  },
  "request_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-26 | Initial | Initial PRD creation |
| 1.6 | 2026-04-08 | Update | Added coordinate and STA update support to PATCH endpoint |

---

**Document Status:** Draft - Pending Review  
**Next Review Date:** TBD  
**Approval Authority:** Bina Marga IT Team Lead