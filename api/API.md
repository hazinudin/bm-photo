# Bina Marga Survey Photo Service - REST API Documentation

**Version:** 1.0.0  
**Base URL:** `https://api.bina-marga.survey`  
**Protocol:** HTTPS (TLS 1.3)

---

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Rate Limiting](#rate-limiting)
4. [Two-Phase Upload Workflow](#two-phase-upload-workflow)
5. [API Endpoints](#api-endpoints)
6. [Error Handling](#error-handling)
7. [Examples](#examples)

---

## Overview

The Bina Marga Survey Photo Service provides a REST API for managing survey photographs of Indonesian national routes. The API supports:

- **Two-phase upload workflow** - Secure direct uploads to Google Cloud Storage
- **Photo catalog management** - Browse, search, update, and delete photos
- **Geospatial organization** - Photos organized by route, STA (station value), and lane
- **API key authentication** with scoped permissions

### Base URL

```
Production:  https://api.bina-marga.survey
Local Dev:   http://localhost:8080
```

### Content Types

All API requests and responses use JSON:

```
Content-Type: application/json
```

---

## Authentication

The API uses API Key authentication via the `X-API-Key` header.

### Header Format

```http
X-API-Key: your-api-key-here
```

### API Key Scopes

API keys have one or more scopes that determine access permissions:

| Scope | Description | Endpoints |
|-------|-------------|-----------|
| `read` | Read photos and metadata | `GET /api/v1/photos/*` |
| `write` | Upload and update photos | `POST /api/v1/photos/*`, `PATCH /api/v1/photos/*` |
| `delete` | Delete photos | `DELETE /api/v1/photos/*` |
| `admin` | Manage API keys and all operations | `POST/GET/DELETE /api/v1/admin/*` |

**Scope Hierarchy:** Higher scopes implicitly satisfy lower ones: `admin` > `delete` > `write` > `read`.

### Authentication Errors

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 401 | `MISSING_API_KEY` | No X-API-Key header provided |
| 401 | `INVALID_API_KEY` | API key not found or invalid |
| 401 | `INACTIVE_API_KEY` | API key has been deactivated |
| 401 | `EXPIRED_API_KEY` | API key has expired |
| 403 | `INSUFFICIENT_SCOPE` | API key lacks required scope |

---

## Rate Limiting

Rate limits are enforced per API key to ensure fair usage.

### Limits

| Operation | Limit | Window |
|-----------|-------|--------|
| Signed URL requests | 10 | per minute |
| Upload confirmations | 10 | per minute |
| Browse requests | 100 | per minute |
| Total requests | 100 | per minute |
| Pending uploads | 10 | concurrent per API key |

### Rate Limit Response

When rate limits are exceeded:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED"
}
```

---

## Two-Phase Upload Workflow

The upload process uses a two-phase approach for security and scalability:

### Phase 1: Get Signed URL

**Endpoint:** `POST /api/v1/photos/upload-url`

Client provides all photo metadata upfront. Server validates data, creates a photo record with "pending" status, and returns:
- `signed_url` - GCS signed URL for direct upload (expires in 15 minutes)
- `upload_token` - Token for Phase 2 confirmation
- `photo_id` - Unique photo identifier

### Client Upload to GCS

**Direct PUT to Google Cloud Storage**

Client uploads the file directly to GCS using the signed URL. This bypasses the backend for better performance.

### Phase 2: Confirm Upload

**Endpoint:** `POST /api/v1/photos/confirm`

Client confirms successful GCS upload. Server:
1. Validates the upload token
2. Verifies the file exists in GCS
3. Marks the upload as completed

### Workflow Diagram

```
┌─────────┐                            ┌──────────┐                    ┌─────────┐
│ Client  │ ── 1. POST /upload-url ──▶ │  Server  │                    │         │
│         │ ◀─ 2. signed_url, token ── │          │                    │         │
└─────────┘                            └──────────┘                    │   GCS   │
     │                                       │                         │         │
     │  3. PUT signed_url (binary)           │                         │         │
     ├────────────────────────────────────────────────────────────────▶│         │
     │                                       │                         │         │
     │ ◀─ 4. 200 OK ───────────────────────────────────────────────────│         │
     │                                       │                         └─────────┘
     │                                       │
     │  5. POST /confirm {token}.            │
     ├──────────────────────────────────────▶│
     │                                       │
     │ ◀─ 6. Upload confirmed ───────────────│
```

---

## API Endpoints

### Health Endpoints

#### GET /health
Basic health check (no authentication required).

**Response:**
```json
{
  "status": "ok"
}
```

#### GET /ready
Readiness check including database connectivity (no authentication required).

**Response:**
```json
{
  "status": "ready"
}
```

**Error Response (503):**
```json
{
  "status": "not_ready"
}
```

---

### Upload Endpoints

#### POST /api/v1/photos/upload-url

**Scope Required:** `write`

Phase 1 of the two-phase upload workflow. Generates a signed URL for direct upload to Google Cloud Storage.

**Request Body:**
```json
{
  "file_metadata": {
    "filename": "survey_photo_001.jpg",
    "content_type": "image/jpeg",
    "file_size_bytes": 2048576
  },
  "photo_attributes": {
    "route_id": "NR-001",
    "lane_code": "L1",
    "latitude": -6.2088,
    "longitude": 106.8456,
    "sta_value": 5.2,
    "description": "Road surface damage at kilometer 5",
    "tags": ["damage", "road_surface"]
  }
}
```

**Field Constraints:**

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| `filename` | string | Yes | Non-empty |
| `content_type` | string | Yes | `image/jpeg`, `image/jpg`, or `image/png` |
| `file_size_bytes` | integer | Yes | 1 - 10,485,760 (10MB) |
| `route_id` | string | Yes | Non-empty |
| `lane_code` | string | Yes | Format: `L1-L10` or `R1-R10` |
| `latitude` | number | Yes | -90 to 90 |
| `longitude` | number | Yes | -180 to 180 |
| `sta_value` | number | No | >= 0 |
| `description` | string | No | Optional description |
| `tags` | array | No | Array of strings |

**Response (201 Created):**
```json
{
  "upload_token": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "signed_url": "https://storage.googleapis.com/bm-survey-photos/photos/2026/NR-001/NR-001_2026_L1_a1b2c3d4.jpg?Expires=1705312800&Signature=...",
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2026-04-01T10:45:00Z"
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid field value |
| 400 | `UNSUPPORTED_FORMAT` | Content type not JPEG/PNG |
| 413 | `FILE_TOO_LARGE` | File exceeds 10MB |
| 429 | `QUOTA_EXCEEDED` | Max 10 pending uploads reached |

---

#### POST /api/v1/photos/confirm

**Scope Required:** `write`

Phase 2 of the two-phase upload workflow. Confirms successful upload to GCS.

**Request Body:**
```json
{
  "upload_token": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
}
```

**Response (200 OK):**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Upload confirmed successfully"
}
```

**Error Responses:**

| Status | Code | Description |
|--------|------|-------------|
| 400 | `INVALID_TOKEN` | Invalid token format |
| 404 | `TOKEN_NOT_FOUND` | Token does not exist |
| 409 | `TOKEN_ALREADY_USED` | Token already processed |
| 410 | `TOKEN_EXPIRED` | Token expired (> 15 min) |

---

### Photo Endpoints

#### GET /api/v1/photos/{photo_id}

**Scope Required:** `read`

Retrieves complete metadata for a specific photo.

**Path Parameters:**
- `photo_id` (string, required) - UUID format

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
  "description": "Road surface damage at kilometer 5",
  "tags": ["damage", "road_surface"],
  "uploaded_at": "2026-04-01T10:30:00Z",
  "download_url": "https://storage.googleapis.com/..."
}
```

---

#### GET /api/v1/photos/{photo_id}/download

**Scope Required:** `read`

Redirects to a signed download URL for the photo file. The signed URL expires after 60 minutes.

**Path Parameters:**
- `photo_id` (string, required) - UUID format

**Response:**
- `302 Found` - Redirects to signed GCS URL

---

#### GET /api/v1/photos

**Scope Required:** `read`

Browse photos with filtering and pagination.

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `route_id` | string | Yes | - | Filter by route |
| `sta_start` | number | No | - | STA range start (>= 0) |
| `sta_end` | number | No | - | STA range end (>= sta_start) |
| `lane_code` | string | No | - | Lane filter (L1-L10, R1-R10) |
| `page` | integer | No | 1 | Page number |
| `per_page` | integer | No | 20 | Items per page (max 100) |

**Response (200 OK):**
```json
{
  "photos": [
    {
      "photo_id": "550e8400-e29b-41d4-a716-446655440000",
      "route_id": "NR-001",
      "lane_code": "L1",
      "sta_value": 5.2,
      "thumbnail_url": "https://storage.googleapis.com/...",
      "uploaded_at": "2026-04-01T10:30:00Z"
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

---

#### PATCH /api/v1/photos/{photo_id}

**Scope Required:** `write`

Updates photo metadata. Only provided fields are updated.

**Path Parameters:**
- `photo_id` (string, required) - UUID format

**Request Body:**
```json
{
  "description": "Updated description",
  "tags": ["damage", "road_surface", "urgent"],
  "lane_code": "R2"
}
```

**Field Constraints:**
- `lane_code` - Must match format `L1-L10` or `R1-R10`

**Response (200 OK):**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Updated description",
  "tags": ["damage", "road_surface", "urgent"],
  "lane_code": "R2",
  "updated_at": "2026-04-01T11:00:00Z"
}
```

---

#### DELETE /api/v1/photos/{photo_id}

**Scope Required:** `delete`

Deletes a photo. Soft delete by default; use `?hard=true` for permanent deletion.

**Path Parameters:**
- `photo_id` (string, required) - UUID format

**Query Parameters:**
- `hard` (boolean, optional) - If true, permanently deletes from database and GCS

**Response (200 OK):**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "deleted_at": "2026-04-01T12:00:00Z",
  "deletion_type": "soft"
}
```

---

## Error Handling

### Error Response Format

All errors follow a consistent JSON structure:

```json
{
  "error": "Human-readable message",
  "code": "MACHINE_READABLE_CODE",
  "details": "Additional context (optional)"
}
```

### Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `BAD_REQUEST` | Invalid request |
| 400 | `VALIDATION_ERROR` | Field validation failed |
| 400 | `INVALID_PHOTO_ID` | Invalid UUID format |
| 400 | `INVALID_TOKEN` | Invalid upload token |
| 400 | `UNSUPPORTED_FORMAT` | Only JPEG/PNG allowed |
| 401 | `MISSING_API_KEY` | No X-API-Key header |
| 401 | `INVALID_API_KEY` | API key not found |
| 401 | `INACTIVE_API_KEY` | API key inactive |
| 401 | `EXPIRED_API_KEY` | API key expired |
| 403 | `INSUFFICIENT_SCOPE` | Missing required scope |
| 404 | `NOT_FOUND` | Resource not found |
| 404 | `PHOTO_NOT_FOUND` | Photo not found |
| 404 | `TOKEN_NOT_FOUND` | Upload token not found |
| 404 | `FILE_NOT_FOUND` | File not in GCS |
| 409 | `TOKEN_ALREADY_USED` | Upload already confirmed |
| 410 | `TOKEN_EXPIRED` | Upload token expired |
| 413 | `FILE_TOO_LARGE` | File exceeds 10MB |
| 429 | `RATE_LIMIT_EXCEEDED` | Too many requests |
| 429 | `QUOTA_EXCEEDED` | Max pending uploads |
| 500 | `INTERNAL_ERROR` | Server error |

---

## Examples

### Complete Upload Example

#### 1. Get Signed URL

```bash
curl -X POST https://api.bina-marga.survey/api/v1/photos/upload-url \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "file_metadata": {
      "filename": "survey_photo.jpg",
      "content_type": "image/jpeg",
      "file_size_bytes": 1024000
    },
    "photo_attributes": {
      "route_id": "NR-001",
      "lane_code": "L1",
      "latitude": -6.2088,
      "longitude": 106.8456,
      "description": "Road surface survey"
    }
  }'
```

**Response:**
```json
{
  "upload_token": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "signed_url": "https://storage.googleapis.com/...",
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2026-04-01T10:45:00Z"
}
```

#### 2. Upload to GCS

```bash
curl -X PUT "https://storage.googleapis.com/bm-survey-photos/..." \
  -H "Content-Type: image/jpeg" \
  --data-binary "@/path/to/survey_photo.jpg"
```

#### 3. Confirm Upload

```bash
curl -X POST https://api.bina-marga.survey/api/v1/photos/confirm \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "upload_token": "f47ac10b-58cc-4372-a567-0e02b2c3d479"
  }'
```

**Response:**
```json
{
  "photo_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Upload confirmed successfully"
}
```

### Browse Photos Example

```bash
curl "https://api.bina-marga.survey/api/v1/photos?route_id=NR-001&sta_start=0&sta_end=10&lane_code=L1&page=1&per_page=20" \
  -H "X-API-Key: your-api-key"
```

### Download Photo Example

```bash
# The download endpoint returns a redirect to a signed URL
curl -L "https://api.bina-marga.survey/api/v1/photos/550e8400-e29b-41d4-a716-446655440000/download" \
  -H "X-API-Key: your-api-key" \
  -o photo.jpg
```

---

## Data Types

### UUID Format

All IDs (photo_id, upload_token) use UUID v4 format:
```
xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
```

### Date-Time Format

All timestamps use RFC3339 format:
```
2026-04-01T10:30:00Z
```

### Coordinates

Coordinates use EPSG:4326 (WGS84):
- Latitude: -90 to 90
- Longitude: -180 to 180

### Lane Code Format

Lane codes follow the pattern:
- `L1` to `L10` - Left lanes
- `R1` to `R10` - Right lanes

---

## GCS CORS Configuration

For client-side uploads, configure GCS bucket CORS:

```json
[
  {
    "origin": ["https://your-app-domain.com"],
    "method": ["PUT"],
    "responseHeader": ["Content-Type", "Content-Length"],
    "maxAgeSeconds": 3600
  }
]
```

---

## Support

For API key requests, technical support, or issues:
- **Email:** it-support@bina-marga.go.id
- **Documentation:** https://docs.bina-marga.survey

---

## Changelog

### v1.0.0 (2026-04-01)
- Initial API release
- Two-phase upload workflow
- Photo browse, update, delete endpoints
- API key authentication with scopes
