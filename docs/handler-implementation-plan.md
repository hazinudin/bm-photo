# Handler Layer Implementation Plan

**Project:** Bina Marga Survey Photo Service  
**Version:** 1.0  
**Date:** April 1, 2026  
**Status:** Planned

---

## Overview

This document outlines the implementation plan for the HTTP Handler Layer, `cmd/server/main.go`, and the integration test plan. The handler layer bridges HTTP requests to the already-implemented service layer.

## Current State

### Completed ✅
- **Domain Layer**: Value objects, entities, DTOs, errors, constants (with tests)
- **Repository Layer**: All 4 PostgreSQL repositories (with integration tests)
- **Service Layer**: UploadService, PhotoService, AuthService, AuditLogService
- **GCS Client**: Signed URLs, file existence checks, deletion
- **Handler Base**: `base.go` (JSON helpers, error formatters, panic recovery)
- **Handler Middleware**: `middleware.go` (Logging, APIKeyAuth, RequireScope, CORS, RateLimiter, query parsers)
- **Migrations**: 3 migration pairs covering full schema

### Pending ⏳
- Concrete HTTP handlers (upload, photo, health)
- Router setup (mux + middleware chain)
- `cmd/server/main.go` (entry point, dependency injection)
- `internal/config/` (configuration loading)
- `GetNewSignedURL` service method (retry endpoint — deferred)

---

## Architecture

### Handler Layer Structure

```
internal/handler/
├── base.go              # BaseHandler: JSON helpers, error formatting, panic recovery
├── middleware.go        # Middleware: Logging, APIKeyAuth, RequireScope, CORS, RateLimiter
├── upload_handler.go    # Upload endpoints (NEW)
├── photo_handler.go     # Photo CRUD + browse endpoints (NEW)
├── health.go            # Health/readiness endpoints (NEW)
├── router.go            # Mux setup + middleware wiring (NEW)
├── upload_handler_test.go  # Integration tests (NEW)
├── photo_handler_test.go   # Integration tests (NEW)
├── health_test.go          # Integration tests (NEW)
└── router_test.go          # Integration tests (NEW)

cmd/server/
└── main.go              # Application entry point (NEW)

internal/config/
└── config.go            # Configuration loading from env (NEW)
```

### Service Error → HTTP Status Mapping

| Service Error | HTTP Status | Error Code |
|--------------|-------------|------------|
| `service.ErrPhotoNotFound` | 404 | PHOTO_NOT_FOUND |
| `service.ErrPhotoDeleted` | 404 | PHOTO_DELETED |
| `service.ErrInvalidToken` | 400 | INVALID_TOKEN |
| `service.ErrTokenNotFound` | 404 | TOKEN_NOT_FOUND |
| `service.ErrTokenAlreadyUsed` | 409 | TOKEN_ALREADY_USED |
| `service.ErrTokenExpired` | 410 | TOKEN_EXPIRED |
| `service.ErrFileNotFound` | 404 | FILE_NOT_FOUND |
| `service.ErrUploadQuotaExceeded` | 429 | UPLOAD_QUOTA_EXCEEDED |
| `service.ErrUnsupportedFormat` | 400 | UNSUPPORTED_FORMAT |
| `service.ErrFileTooLarge` | 413 | FILE_TOO_LARGE |
| `service.ErrInvalidCoordinates` | 400 | INVALID_COORDINATES |
| `service.ErrInvalidRouteID` | 400 | INVALID_ROUTE_ID |
| `service.ErrInvalidLaneCode` | 400 | INVALID_LANE_CODE |
| `service.ErrInvalidSTAValue` | 400 | INVALID_STA_VALUE |
| `service.ErrUnauthorized` | 401 | UNAUTHORIZED |
| `service.ErrForbidden` | 403 | FORBIDDEN |
| `service.ErrScopeRead` | 403 | INSUFFICIENT_SCOPE |
| `service.ErrScopeWrite` | 403 | INSUFFICIENT_SCOPE |
| `service.ErrScopeAdmin` | 403 | INSUFFICIENT_SCOPE |
| `model.ValidationError` | 400 | VALIDATION_ERROR |
| `service.ServiceError` (other) | 500 | INTERNAL_ERROR |

---

## Implementation Details

### 1. `internal/handler/upload_handler.go`

**Struct:**
```go
type UploadHandler struct {
    *BaseHandler
    uploadSvc service.UploadService
}

func NewUploadHandler(uploadSvc service.UploadService, logger *slog.Logger) *UploadHandler
```

#### 1.1 POST `/api/v1/photos/upload-url` — `GetSignedUploadURL`

**Flow:**
1. Set response header `Content-Type: application/json`
2. Decode JSON body into `rest.GetSignedUploadURLRequest`
3. Call `uploadSvc.GetSignedURL(ctx, &req, handler.GetAPIKeyID(r))`
4. Map errors to HTTP status codes (see mapping table above)
5. Return 201 with `GetSignedUploadURLResponse`

**Error mapping:**
- JSON decode error → 400 BAD_REQUEST
- `ValidationError` → 400 (via `handleValidationError`)
- `ErrUploadQuotaExceeded` → 429
- `ErrUnsupportedFormat` → 400
- `ErrFileTooLarge` → 413
- `ErrInvalidCoordinates` → 400
- `ErrInvalidRouteID` → 400
- `ErrInvalidLaneCode` → 400
- `service.ServiceError` → 500

#### 1.2 POST `/api/v1/photos/confirm` — `ConfirmUpload`

**Flow:**
1. Decode JSON body into `rest.ConfirmUploadRequest`
2. Validate token format
3. Call `uploadSvc.ConfirmUpload(ctx, req.UploadToken, apiKeyID)`
4. Map errors to HTTP status codes
5. Return 200 with `ConfirmUploadResponse`

**Error mapping:**
- JSON decode error → 400
- `ErrInvalidToken` → 400
- `ErrTokenNotFound` → 404
- `ErrTokenAlreadyUsed` → 409
- `ErrTokenExpired` → 410
- `ErrFileNotFound` → 404
- `service.ServiceError` → 500

#### 1.3 POST `/api/v1/photos/{photo_id}/new-signed-url` — `GetNewSignedURL` ⏳ DEFERRED

**Status:** Skipped for now. The service method `UploadService.GetNewSignedURL` does not exist yet. This handler will be implemented when the service method is ready.

**TODO:** Add to service layer plan as well.

---

### 2. `internal/handler/photo_handler.go`

**Struct:**
```go
type PhotoHandler struct {
    *BaseHandler
    photoSvc  service.PhotoService
    gcsClient service.GCSClient
}

func NewPhotoHandler(photoSvc service.PhotoService, gcsClient service.GCSClient, logger *slog.Logger) *PhotoHandler
```

#### 2.1 GET `/api/v1/photos/{photo_id}` — `GetPhoto`

**Flow:**
1. Extract `photo_id` from path via `r.PathValue("photo_id")`
2. Parse to `vo.PhotoID` — if invalid, return 400
3. Call `photoSvc.GetByID(ctx, photoID)`
4. Generate download URL: `service.GenerateDownloadURL(gcsClient, photo.GCSObjectName(), 60)`
5. Build response: `service.BuildPhotoResponse(photo, downloadURL)`
6. Return 200

**Error mapping:**
- Invalid photo_id format → 400
- `ErrPhotoNotFound` → 404
- `ErrPhotoDeleted` → 404
- `service.ServiceError` → 500

#### 2.2 GET `/api/v1/photos` — `BrowsePhotos`

**Flow:**
1. Parse query params using middleware helpers:
   - `route_id` → `r.URL.Query().Get("route_id")`
   - `sta_start` → `ParseQueryFloat64(r, "sta_start")`
   - `sta_end` → `ParseQueryFloat64(r, "sta_end")`
   - `lane_code` → `ParseQueryString(r, "lane_code")`
   - `page` → `ParseQueryInt(r, "page", model.DefaultPage)`
   - `per_page` → `ParseQueryInt(r, "per_page", model.DefaultPerPage)`
2. Build `rest.BrowsePhotosRequest`, call `Validate()`
3. Convert to `repository.BrowseFilter`
4. Call `photoSvc.Browse(ctx, filter)`
5. Return 200 with `BrowsePhotosResponse`

**Error mapping:**
- Missing `route_id` → 400
- `sta_start > sta_end` → 400
- Invalid `lane_code` → 400
- `service.ServiceError` → 500

#### 2.3 PATCH `/api/v1/photos/{photo_id}` — `UpdatePhoto`

**Flow:**
1. Extract `photo_id` from path
2. Decode JSON body into `rest.UpdatePhotoRequest`
3. Validate
4. Call `photoSvc.Update(ctx, photoID, &req)`
5. Return 200 with `UpdatePhotoResponse`

**Error mapping:**
- Invalid photo_id → 400
- JSON decode error → 400
- `ValidationError` → 400
- `ErrPhotoNotFound` → 404
- `ErrPhotoDeleted` → 404
- `service.ServiceError` → 500

#### 2.4 DELETE `/api/v1/photos/{photo_id}` — `DeletePhoto`

**Flow:**
1. Extract `photo_id` from path
2. Check query param `hard` → `r.URL.Query().Get("hard") == "true"`
3. Call `photoSvc.Delete(ctx, photoID, hard, apiKeyID)`
4. Return 200 with `DeletePhotoResponse`

**Error mapping:**
- Invalid photo_id → 400
- `ErrPhotoNotFound` → 404
- `ErrPhotoDeleted` → 404
- `service.ServiceError` → 500

#### 2.5 GET `/api/v1/photos/{photo_id}/download` — `DownloadPhoto`

**Flow:**
1. Extract `photo_id` from path
2. Call `photoSvc.GetByID(ctx, photoID)`
3. Generate signed download URL: `gcsClient.GenerateSignedURL(photo.GCSObjectName(), photo.FileFormat().ContentType(), 15)`
4. Return 302 redirect to signed URL

**Error mapping:**
- Invalid photo_id → 400
- `ErrPhotoNotFound` → 404
- `ErrPhotoDeleted` → 404
- GCS error → 500

---

### 3. `internal/handler/health.go`

```go
package handler

import (
    "context"
    "net/http"
    "time"
)

type HealthHandler struct {
    dbPinger interface {
        Ping(ctx context.Context) error
    }
}

func NewHealthHandler(dbPinger interface{ Ping(ctx context.Context) error }) *HealthHandler

func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request)
func (h *HealthHandler) ReadinessCheck(w http.ResponseWriter, r *http.Request)
```

**`HealthCheck`:** Returns 200 `{"status": "ok"}` — no auth required.

**`ReadinessCheck`:** Pings database with 2-second timeout. Returns 200 `{"status": "ready"}` if successful, 503 `{"status": "not_ready"}` if DB is unreachable.

---

### 4. `internal/handler/router.go`

```go
func NewRouter(
    uploadSvc service.UploadService,
    photoSvc service.PhotoService,
    authSvc service.AuthService,
    gcsClient service.GCSClient,
    dbPinger interface{ Ping(ctx context.Context) error },
    logger *slog.Logger,
) http.Handler
```

**Setup:**
```go
mux := http.NewServeMux()

// Health endpoints (no auth)
healthHandler := NewHealthHandler(dbPinger)
mux.HandleFunc("GET /health", healthHandler.HealthCheck)
mux.HandleFunc("GET /ready", healthHandler.ReadinessCheck)

// Handlers
uploadHandler := NewUploadHandler(uploadSvc, logger)
photoHandler := NewPhotoHandler(photoSvc, gcsClient, logger)
mw := NewMiddleware(authSvc, logger)

// Authenticated routes with middleware chain
// Order: Logging → CORS → RateLimit → APIKeyAuth → RequireScope → handler

// Upload routes (write scope)
uploadMux := http.NewServeMux()
uploadMux.HandleFunc("POST /api/v1/photos/upload-url", uploadHandler.GetSignedUploadURL)
uploadMux.HandleFunc("POST /api/v1/photos/confirm", uploadHandler.ConfirmUpload)
// TODO: uploadMux.HandleFunc("POST /api/v1/photos/{photo_id}/new-signed-url", uploadHandler.GetNewSignedURL)

// Photo read routes (read scope)
readMux := http.NewServeMux()
readMux.HandleFunc("GET /api/v1/photos/{photo_id}", photoHandler.GetPhoto)
readMux.HandleFunc("GET /api/v1/photos/{photo_id}/download", photoHandler.DownloadPhoto)
readMux.HandleFunc("GET /api/v1/photos", photoHandler.BrowsePhotos)

// Photo write routes (write scope)
writeMux := http.NewServeMux()
writeMux.HandleFunc("PATCH /api/v1/photos/{photo_id}", photoHandler.UpdatePhoto)

// Photo admin routes (admin scope)
adminMux := http.NewServeMux()
adminMux.HandleFunc("DELETE /api/v1/photos/{photo_id}", photoHandler.DeletePhoto)

// Apply middleware chains
handler := http.NewServeMux()
handler.Handle("/api/v1/photos/upload-url", mw.Logging(mw.CORS(mw.RateLimiter.Middleware(mw.APIKeyAuth(mw.RequireScope("write")(uploadMux))))))
handler.Handle("/api/v1/photos/confirm", mw.Logging(mw.CORS(mw.RateLimiter.Middleware(mw.APIKeyAuth(mw.RequireScope("write")(uploadMux))))))
handler.Handle("/api/v1/photos/{photo_id}", mw.Logging(mw.CORS(mw.RateLimiter.Middleware(mw.APIKeyAuth(mw.RequireScope("read")(readMux))))))
handler.Handle("/api/v1/photos/{photo_id}/download", mw.Logging(mw.CORS(mw.RateLimiter.Middleware(mw.APIKeyAuth(mw.RequireScope("read")(readMux))))))
handler.Handle("/api/v1/photos", mw.Logging(mw.CORS(mw.RateLimiter.Middleware(mw.APIKeyAuth(mw.RequireScope("read")(readMux))))))
handler.Handle("/api/v1/photos/{photo_id}", mw.Logging(mw.CORS(mw.RateLimiter.Middleware(mw.APIKeyAuth(mw.RequireScope("write")(writeMux))))))
handler.Handle("/api/v1/photos/{photo_id}", mw.Logging(mw.CORS(mw.RateLimiter.Middleware(mw.APIKeyAuth(mw.RequireScope("admin")(adminMux))))))

// Health endpoints (no middleware)
handler.HandleFunc("GET /health", healthHandler.HealthCheck)
handler.HandleFunc("GET /ready", healthHandler.ReadinessCheck)

return handler
```

**Note:** The middleware chaining above is verbose. A cleaner approach would be a `Chain` helper or per-route middleware application. The actual implementation should use a cleaner pattern — see code style below.

---

### 5. `internal/config/config.go`

```go
package config

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    GCS      GCSConfig
}

type ServerConfig struct {
    Port     int
    LogLevel string
}

type DatabaseConfig struct {
    Host     string
    Port     int
    Name     string
    Username string
    Password string
    MaxConns int32
    MinConns int32
}

type GCSConfig struct {
    BucketName            string
    CredentialsFile       string
    SignedURLExpiryMinutes int
}

func Load() (*Config, error)
```

**Loading strategy:** Read from environment variables using `os.Getenv`. The `.env` file already has all needed vars. For local development, use `godotenv` or just source the `.env` file before running.

**Environment variables:**
| Env Var | Config Field | Default |
|---------|-------------|---------|
| `PORT` | Server.Port | 8080 |
| `LOG_LEVEL` | Server.LogLevel | "info" |
| `DB_HOST` | Database.Host | "localhost" |
| `DB_PORT` | Database.Port | 5432 |
| `DB_NAME` | Database.Name | "bm_photos" |
| `DB_USERNAME` | Database.Username | "postgres" |
| `DB_PASSWORD` | Database.Password | "" |
| `GCS_BUCKET_NAME` | GCS.BucketName | required |
| `GOOGLE_APPLICATION_CREDENTIALS` | GCS.CredentialsFile | "" |

---

### 6. `cmd/server/main.go`

**Flow:**
1. Load config via `config.Load()`
2. Initialize structured logger (`slog`) with configured log level
3. Connect to PostgreSQL via `pgxpool`
4. Run database migrations via `internal/database/migrate.go`
5. Initialize GCS client via `client/gcs.New()`
6. Initialize repositories (PostgreSQL implementations)
7. Initialize services (UploadService, PhotoService, AuthService, AuditLogService)
8. Build router via `handler.NewRouter(...)`
9. Start HTTP server with graceful shutdown

```go
func main() {
    // 1. Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // 2. Initialize logger
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: parseLogLevel(cfg.Server.LogLevel),
    }))

    // 3. Connect to database
    ctx := context.Background()
    pool, err := connectDatabase(ctx, cfg.Database)
    if err != nil {
        logger.Fatal("failed to connect to database", "error", err)
    }
    defer pool.Close()

    // 4. Run migrations
    if err := runMigrations(cfg.Database); err != nil {
        logger.Fatal("failed to run migrations", "error", err)
    }

    // 5. Initialize GCS client
    gcsClient, err := gcs.NewClient(ctx, cfg.GCS)
    if err != nil {
        logger.Fatal("failed to initialize GCS client", "error", err)
    }
    defer gcsClient.Close()

    // 6. Initialize repositories
    photoRepo := postgres.NewPhotoRepository(pool)
    pendingUploadRepo := postgres.NewPendingUploadRepository(pool)
    apiKeyRepo := postgres.NewAPIKeyRepository(pool)
    auditLogRepo := postgres.NewAuditLogRepository(pool)

    // 7. Initialize services
    auditSvc := service.NewAuditLogService(auditLogRepo, logger)
    uploadSvc := service.NewUploadService(photoRepo, pendingUploadRepo, gcsClient, logger)
    photoSvc := service.NewPhotoService(photoRepo, gcsClient, logger, auditSvc)
    authSvc := service.NewAuthService(apiKeyRepo, logger)

    // 8. Build router
    router := handler.NewRouter(uploadSvc, photoSvc, authSvc, gcsClient, pool, logger)

    // 9. Start HTTP server
    addr := fmt.Sprintf(":%d", cfg.Server.Port)
    logger.Info("starting server", "addr", addr)

    srv := &http.Server{
        Addr:         addr,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        <-sigCh
        logger.Info("shutting down server...")
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        if err := srv.Shutdown(shutdownCtx); err != nil {
            logger.Error("server shutdown failed", "error", err)
        }
    }()

    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Fatal("server failed", "error", err)
    }
}
```

---

## Integration Test Plan

### Test Infrastructure

**Database:** Use the existing PostgreSQL database from `.env` (`bm_photos_test`). Tests will use the real database — migrations will be applied before test runs.

**GCS Client:** Use the real GCS client. Tests will upload to a dedicated test prefix in the GCS bucket (configured via `GCS_TEST_PREFIX` in `.env`). All test objects will be cleaned up after each test run.

**Test Server:** Use `httptest.NewServer` with the real router + services + real GCS client.

**Test Data Setup:** Each test will seed its own data via repository calls, and clean up after itself via `t.Cleanup()`.

### Test Helpers

```go
// testServer wraps httptest.Server with test dependencies
type testServer struct {
    *httptest.Server
    photoRepo         repository.PhotoRepository
    pendingUploadRepo repository.PendingUploadRepository
    apiKeyRepo        repository.APIKeyRepository
    gcsClient         service.GCSClient
}

// setupTestServer creates a test server with real DB and real GCS
func setupTestServer(t *testing.T) *testServer

// createTestAPIKey inserts an API key for testing and returns the raw key
func createTestAPIKey(t *testing.T, repo repository.APIKeyRepository, scopes []string) string

// doRequest performs an HTTP request and returns the response
func doRequest(t *testing.T, server *httptest.Server, method, path string, body io.Reader, apiKey string) *http.Response

// cleanupGCSObject deletes a test object from GCS
func cleanupGCSObject(t *testing.T, gcsClient service.GCSClient, objectName string)
```

### Test Seed Data

Each test suite needs:
1. **API Key** with appropriate scopes:
   - `test-api-key-read` — scopes: `["read"]`
   - `test-api-key-write` — scopes: `["read", "write"]`
   - `test-api-key-admin` — scopes: `["read", "write", "admin"]`
2. **Photos** (for browse/get/update/delete tests):
   - Completed photos with various route_id, lane_code, sta_value
   - Pending photos (for retry tests — when retry endpoint is implemented)
   - Soft-deleted photos
3. **GCS Objects**: Test files uploaded to the `GCS_TEST_PREFIX` path in the bucket, cleaned up after each test

### Test Cases

#### Upload Handler Tests (`upload_handler_test.go`)

| # | Test Name | Scenario | Expected |
|---|-----------|----------|----------|
| 1 | `TestGetSignedUploadURL_ValidRequest_Returns201` | Valid body, write scope API key | 201, response with signed_url, upload_token, photo_id, expires_at |
| 2 | `TestGetSignedUploadURL_MissingAPIKey_Returns401` | No X-API-Key header | 401, `{"error": "missing API key", "code": "MISSING_API_KEY"}` |
| 3 | `TestGetSignedUploadURL_InvalidAPIKey_Returns401` | Wrong API key value | 401, `{"error": "invalid API key", "code": "INVALID_API_KEY"}` |
| 4 | `TestGetSignedUploadURL_ReadScopeOnly_Returns403` | API key with only "read" scope | 403, `{"error": "insufficient scope", "code": "INSUFFICIENT_SCOPE"}` |
| 5 | `TestGetSignedUploadURL_MalformedJSON_Returns400` | Invalid JSON body | 400 |
| 6 | `TestGetSignedUploadURL_MissingFilename_Returns400` | Empty filename in file_metadata | 400, VALIDATION_ERROR |
| 7 | `TestGetSignedUploadURL_MissingContentType_Returns400` | Empty content_type | 400, VALIDATION_ERROR |
| 8 | `TestGetSignedUploadURL_FileTooLarge_Returns400` | file_size_bytes > 10MB | 400 (validation catches it) |
| 9 | `TestGetSignedUploadURL_UnsupportedFormat_Returns400` | content_type = "image/gif" | 400 |
| 10 | `TestGetSignedUploadURL_InvalidLaneCode_Returns400` | lane_code = "X99" | 400, VALIDATION_ERROR |
| 11 | `TestGetSignedUploadURL_InvalidLatitude_Returns400` | latitude = 200 | 400, VALIDATION_ERROR |
| 12 | `TestGetSignedUploadURL_InvalidLongitude_Returns400` | longitude = 300 | 400, VALIDATION_ERROR |
| 13 | `TestGetSignedUploadURL_NegativeSTAValue_Returns400` | sta_value = -1 | 400, VALIDATION_ERROR |
| 14 | `TestGetSignedUploadURL_QuotaExceeded_Returns429` | 10 pending uploads already | 429, UPLOAD_QUOTA_EXCEEDED |
| 15 | `TestConfirmUpload_ValidToken_Returns200` | Valid token, file uploaded to real GCS | 200, `{"photo_id": "...", "message": "Upload confirmed successfully"}` |
| 16 | `TestConfirmUpload_MalformedJSON_Returns400` | Invalid JSON body | 400 |
| 17 | `TestConfirmUpload_InvalidTokenFormat_Returns400` | upload_token = "not-a-uuid" | 400 |
| 18 | `TestConfirmUpload_TokenNotFound_Returns404` | Non-existent token UUID | 404 |
| 19 | `TestConfirmUpload_TokenAlreadyUsed_Returns409` | Token already confirmed | 409 |
| 20 | `TestConfirmUpload_TokenExpired_Returns410` | Token with past expires_at | 410 |
| 21 | `TestConfirmUpload_FileNotInGCS_Returns404` | Token valid but file not uploaded to GCS | 404 |
| 22 | `TestConfirmUpload_WrongAPIKey_Returns404` | Different API key than creator | 404 |

#### Photo Handler Tests (`photo_handler_test.go`)

| # | Test Name | Scenario | Expected |
|---|-----------|----------|----------|
| 1 | `TestGetPhoto_ValidID_Returns200` | Existing completed photo | 200, full metadata + download_url |
| 2 | `TestGetPhoto_NotFound_Returns404` | Non-existent photo_id | 404 |
| 3 | `TestGetPhoto_DeletedPhoto_Returns404` | Soft-deleted photo | 404 |
| 4 | `TestGetPhoto_InvalidID_Returns400` | Malformed UUID in path | 400 |
| 5 | `TestBrowsePhotos_ByRoute_Returns200` | route_id with seeded photos | 200, paginated list with photos array |
| 6 | `TestBrowsePhotos_ByRouteAndSTA_Returns200` | route_id + sta_start + sta_end | 200, filtered to STA range |
| 7 | `TestBrowsePhotos_ByRouteAndLane_Returns200` | route_id + lane_code | 200, filtered to lane |
| 8 | `TestBrowsePhotos_MissingRouteID_Returns400` | No route_id query param | 400 |
| 9 | `TestBrowsePhotos_InvalidSTARange_Returns400` | sta_start > sta_end | 400 |
| 10 | `TestBrowsePhotos_InvalidLaneCode_Returns400` | lane_code = "X99" | 400 |
| 11 | `TestBrowsePhotos_Pagination_Defaults` | No page/per_page params | Uses defaults (page=1, per_page=20) |
| 12 | `TestBrowsePhotos_Pagination_MaxPerPage` | per_page=200 | Capped at 100 |
| 13 | `TestBrowsePhotos_NoResults_ReturnsEmptyList` | route_id with no photos | 200, empty photos array, total_count=0 |
| 14 | `TestBrowsePhotos_ReadScopeOnly_Returns200` | API key with only "read" scope | 200 |
| 15 | `TestUpdatePhoto_ValidUpdate_Returns200` | Update description + tags | 200, updated fields |
| 16 | `TestUpdatePhoto_InvalidLaneCode_Returns400` | lane_code = "X99" | 400 |
| 17 | `TestUpdatePhoto_PhotoNotFound_Returns404` | Non-existent photo | 404 |
| 18 | `TestUpdatePhoto_ReadScopeOnly_Returns403` | API key with only "read" | 403 |
| 19 | `TestDeletePhoto_SoftDelete_Returns200` | DELETE without hard param, admin scope | 200, deletion_type="soft" |
| 20 | `TestDeletePhoto_HardDelete_Returns200` | DELETE ?hard=true, admin scope | 200, deletion_type="hard", GCS file actually deleted |
| 21 | `TestDeletePhoto_AlreadySoftDeleted_Returns404` | Soft-deleted photo, soft delete again | 404 |
| 22 | `TestDeletePhoto_ReadScopeOnly_Returns403` | API key without admin scope | 403 |
| 23 | `TestDeletePhoto_WriteScopeOnly_Returns403` | API key with write but not admin | 403 |
| 24 | `TestDownloadPhoto_ValidID_Returns302` | Existing photo | 302, Location header with signed URL |
| 25 | `TestDownloadPhoto_NotFound_Returns404` | Non-existent photo | 404 |
| 26 | `TestDownloadPhoto_DeletedPhoto_Returns404` | Soft-deleted photo | 404 |

#### Health Handler Tests (`health_test.go`)

| # | Test Name | Scenario | Expected |
|---|-----------|----------|----------|
| 1 | `TestHealthCheck_Returns200` | GET /health | 200, `{"status": "ok"}` |
| 2 | `TestHealthCheck_NoAuthRequired` | No API key header | 200 (no auth needed) |
| 3 | `TestReadinessCheck_DBUp_Returns200` | DB reachable | 200, `{"status": "ready"}` |
| 4 | `TestReadinessCheck_DBDown_Returns503` | Mock pinger returns error | 503, `{"status": "not_ready"}` |

#### Router/Middleware Tests (`router_test.go`)

| # | Test Name | Scenario | Expected |
|---|-----------|----------|----------|
| 1 | `TestRouter_CORS_Preflight` | OPTIONS request to any endpoint | 204, CORS headers present |
| 2 | `TestRouter_CORS_HeadersOnResponse` | GET request | Response has Access-Control-Allow-Origin header |
| 3 | `TestRouter_UnknownRoute_Returns404` | GET /api/v1/nonexistent | 404 |
| 4 | `TestRouter_MethodNotAllowed` | PUT /api/v1/photos | 404 (Go 1.22 mux returns 404 for unmatched methods) |
| 5 | `TestRouter_RateLimit_NotExceeded` | 50 requests in window | All succeed |
| 6 | `TestRouter_MiddlewareChain_Order` | Request without API key to protected endpoint | 401 (auth checked before scope) |

### Test Execution

```bash
# Run all handler tests
go test -v ./internal/handler/...

# Run with database (requires running PostgreSQL)
# Set env vars from .env
go test -v ./internal/handler/... -count=1

# Run specific test
go test -v -run TestGetSignedUploadURL_ValidRequest_Returns201 ./internal/handler/
```

### Test Cleanup Strategy

- Each test creates its own data with unique identifiers (UUIDs)
- Tests clean up after themselves via `t.Cleanup()`
- Use transactions where possible for automatic rollback
- For the `bm_photos_test` database, run a cleanup script before each test run to remove orphaned test data

---

## Implementation Order

1. **`internal/config/config.go`** — Configuration loading (needed by main.go)
2. **`internal/handler/upload_handler.go`** — Upload endpoints (except retry)
3. **`internal/handler/photo_handler.go`** — Photo CRUD + browse endpoints
4. **`internal/handler/health.go`** — Health/readiness endpoints
5. **`internal/handler/router.go`** — Mux setup + middleware wiring
6. **`cmd/server/main.go`** — Entry point, DI, server startup
7. **`internal/handler/upload_handler_test.go`** — Upload tests
8. **`internal/handler/photo_handler_test.go`** — Photo tests
9. **`internal/handler/health_test.go`** — Health tests
10. **`internal/handler/router_test.go`** — Router/middleware tests

---

## Deferred Items

| Item | Reason | Where Tracked |
|------|--------|---------------|
| `GetNewSignedURL` handler + service method | Service method doesn't exist yet | This doc + service-layer-plan.md |
| gRPC handlers | Lower priority, internal only | Future task |
| LRS client integration | Deferred per PRD | Future task |
| OpenAPI spec | Documentation, not blocking | Future task |
| Makefile, Dockerfile, docker-compose | Infra, not blocking | Future task |

---

## Success Criteria

- [ ] All 8 REST endpoints implemented (excluding retry)
- [ ] All endpoints return correct HTTP status codes and JSON responses
- [ ] Middleware chain works correctly (auth, scope, CORS, rate limiting, logging)
- [ ] Health and readiness endpoints work
- [ ] `cmd/server/main.go` starts the server successfully
- [ ] All integration tests pass against real PostgreSQL
- [ ] No lint errors (`golangci-lint run`)
- [ ] Code follows AGENTS.md conventions (import ordering, naming, error handling)
