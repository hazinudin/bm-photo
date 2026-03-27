# AGENTS.md - Bina Marga Survey Photo Service

This file contains instructions for AI coding agents working on this codebase.

## Project Overview

Bina Marga Survey Photo Service is a photo catalog system for managing survey photographs of Indonesian national routes. See `PRD.md` for detailed requirements.

**Tech Stack:**
- Go 1.22+ with standard library `net/http` (no web framework)
- gRPC for internal microservice communication
- PostgreSQL 15+ with pgx v5 driver
- golang-migrate/migrate v4 for database migrations
- Google Cloud Storage for photo storage
- Prometheus + Grafana for monitoring
- Docker for containerization

## Build Commands

```bash
# Build the service
go build -o bin/server ./cmd/server

# Build with version info
go build -ldflags "-X main.version=$(git describe --tags)" -o bin/server ./cmd/server

# Run the service locally
go run ./cmd/server

# Run with specific config
go run ./cmd/server --config config/local.yaml
```

## Lint Commands

```bash
# Run golangci-lint (install first: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
golangci-lint run

# Run with specific linters
golangci-lint run --enable=errcheck,gosimple,govet,staticcheck,ineffassign

# Format code
go fmt ./...

# Format and simplify
gofmt -s -w .
```

## Test Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run tests with coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Run a single test file
go test -v -run TestFunctionName ./path/to/package

# Run a single test by name pattern
go test -v -run "Test.*Upload" ./...

# Run tests for a specific package
go test -v ./internal/service/photo

# Run integration tests (requires test database)
go test -v -tags=integration ./...

# Run benchmark tests
go test -bench=. ./...
```

## Database Migrations

```bash
# Install migrate tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create a new migration
migrate create -ext sql -dir migrations -seq add_new_table

# Apply migrations (up)
migrate -database "postgres://user:pass@localhost:5432/bm_photos?sslmode=disable" -path migrations up

# Rollback last migration (down 1)
migrate -database "postgres://user:pass@localhost:5432/bm_photos?sslmode=disable" -path migrations down 1

# Check current migration version
migrate -database "postgres://user:pass@localhost:5432/bm_photos?sslmode=disable" -path migrations version

# Force migration version (use with caution)
migrate -database "postgres://..." -path migrations force VERSION

# Apply migrations programmatically (in code)
# See: internal/database/migrate.go
```

## Code Style Guidelines

### Import Ordering

Imports must be grouped in the following order, separated by blank lines:

```go
// 1. Standard library
import (
    "context"
    "net/http"
    "time"

// 2. Third-party packages
    "github.com/jackc/pgx/v5"
    "github.com/prometheus/client_golang/prometheus"
    "google.golang.org/grpc"

// 3. Internal packages
    "github.com/bina-marga/survey-photo/internal/config"
    "github.com/bina-marga/survey-photo/internal/service"
    "github.com/bina-marga/survey-photo/internal/repository"
)
```

### File Structure

Source files use `.go` extension. Filenames should be `snake_case.go`. Proto files use `.proto` extension.

### Formatting

- Use `gofmt` for formatting (auto-format on save in IDE)
- Use `gofmt -s` to simplify code
- Max line length: 120 characters (configure in editor)
- Use tabs for indentation (Go standard)

### Types and Interfaces

```go
// Prefer interfaces for external dependencies and mocking
type GCSClient interface {
    Upload(ctx context.Context, objectName string, data []byte) error
    GenerateSignedURL(objectName string, expiry time.Duration) (string, error)
}

// Use strong types, avoid interface{} where possible
type PhotoID string // Strong type for photo identifiers

// Use struct tags for JSON and database mapping
type Photo struct {
    IDPhotoID`json:"photo_id" db:"photo_id"`
    RouteID    string    `json:"route_id" db:"route_id"`
    LaneNumber int`json:"lane_number" db:"lane_number"`
    Latitude   float64   `json:"latitude" db:"latitude"`
    Longitude  float64   `json:"longitude" db:"longitude"`
    STAValue   float64   `json:"sta_value" db:"sta_value"`
    CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Use pointers for optional fields
type PhotoUpdate struct {
    Description *string`json:"description,omitempty"`
    Tags        *[]string `json:"tags,omitempty"`
    LaneNumber  *int      `json:"lane_number,omitempty"`
}
```

### Naming Conventions

- **Packages:** Lowercase, single word preferred (`service`, `repository`, `handler`)
- **Files:** `snake_case.go` (e.g., `photo_service.go`, `upload_handler.go`)
- **Types/Interfaces:** `PascalCase` (e.g., `PhotoService`, `UploadRepository`)
- **Functions/Methods:** `PascalCase` (exported), `camelCase` (unexported)
- **Constants:** `PascalCase` or `UPPER_SNAKE_CASE` for truly constant values
- **Errors:** Use `Err` prefix for error variables (e.g., `ErrPhotoNotFound`)

### Error Handling

```go
// Define sentinel errors
var (
    ErrPhotoNotFound     = errors.New("photo not found")
    ErrUploadTokenExpired = errors.New("upload token has expired")
    ErrInvalidAPIKey      = errors.New("invalid API key")
)

// Wrap errors with context
func (s *Service) GetPhoto(ctx context.Context, id string) (*Photo, error) {
    photo, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get photo %s: %w", id, err)
    }
    return photo, nil
}

// Use errors.Is and errors.As for checking
if errors.Is(err, ErrPhotoNotFound) {
    return http.StatusNotFound
}

// Create custom error types for structured errors
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
```

### Logging

```go
// Use structured logging (zerolog or zap)
// Example with zerolog:
import "github.com/rs/zerolog/log"

func (h *Handler) UploadURL(w http.ResponseWriter, r *http.Request) {
    logger := log.With().
        Str("request_id", getRequestID(r)).
        Str("api_key_id", getAPIKeyID(r)).
        Logger()

    logger.Info().Msg("Processing upload URL request")

    // ... handler logic

    logger.Info().
        Str("photo_id", photoID).
        Dur("duration", time.Since(start)).
        Msg("Upload URL generated successfully")
}

// Log levels:
// ERROR - Application errors, service failures
// WARN  - Deprecated usage, recoverable issues
// INFO  - Normal operations (startups, shutdowns, important events)
// DEBUG - Detailed information (disabled in production)
```

### Testing Conventions

```go
// Test file naming: <name>_test.go (same directory as source file)

// Test function naming: Test<FunctionName><Scenario>
func TestUploadURL_ValidRequest_ReturnsSignedURL(t *testing.T) {}

func TestUploadURL_InvalidAPIKey_ReturnsUnauthorized(t *testing.T) {}

func TestUploadURL_ExpiredToken_ReturnsGone(t *testing.T) {}

// Use table-driven tests for multiple scenarios
func TestValidatePhotoMetadata(t *testing.T) {
    tests := []struct {
        name    string
        input   PhotoMetadata
        wantErr error
    }{
        {
            name: "valid metadata",
            input: PhotoMetadata{
                RouteID:    "NR-001",
                LaneNumber: 1,
                Latitude:   -6.2088,
                Longitude:  106.8456,
            },
            wantErr: nil,
        },
        {
            name: "invalid latitude",
            input: PhotoMetadata{
                RouteID:    "NR-001",
                LaneNumber: 1,
                Latitude:   -100.0, // invalid
                Longitude:  106.8456,
            },
            wantErr: &ValidationError{Field: "latitude"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePhotoMetadata(tt.input)
            if tt.wantErr != nil {
                assert.Error(t, err)
                assert.EqualError(t, err, tt.wantErr.Error())
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

// Use testify/assert for assertions
import "github.com/stretchr/testify/assert"

// Mock interfaces for external dependencies
//go:generate mockgen -source=interface.go -destination=mock/mock.go
```

### Context Usage

```go
// Always pass context as the first parameter to functions that need it
func (s *Service) CreatePhoto(ctx context.Context, req *CreatePhotoRequest) error {}

// Use context for cancellation and timeouts
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
defer cancel()

// Propagate context through call chains
result, err := s.repo.GetPhoto(ctx, id)
```

### HTTP Handlers

```go
// Use standard library net/http
// Handler pattern:
func (h *Handler) GetPhoto(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Extract path parameters using Go 1.22+ path values
    photoID := r.PathValue("id")

    // Validate input
    if photoID == "" {
        http.Error(w, "photo_id is required", http.StatusBadRequest)
        return
    }

    // Call service
    photo, err := h.service.GetPhoto(ctx, photoID)
    if errors.Is(err, ErrPhotoNotFound) {
        http.Error(w, "photo not found", http.StatusNotFound)
        return
    }
    if err != nil {
        log.Ctx(ctx).Error().Err(err).Msg("failed to get photo")
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    // Write response
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(photo); err != nil {
        log.Ctx(ctx).Error().Err(err).Msg("failed to encode response")
    }
}
```

## Project Structure

```
bm-photo/
├── cmd/
│   └── server/main.go# Application entry point
├── internal/
│   ├── config/config.go         # Configuration loading
│   ├── handler/                  # HTTP handlers (REST)
│   │   ├── photo.go
│   │   ├── health.go
│   │   └── middleware.go
│   ├── grpc/                    # gRPC handlers
│   │   └── photo_catalog.go
│   ├── service/                  # Business logic
│   │   └── photo.go
│   ├── repository/               # Database operations
│   │   ├── photo.go
│   │   └── pending_upload.go
│   ├── client/                   # External service clients
│   │   ├── gcs.go
│   │   └── lrs.go
│   ├── model/                    # Domain models and DTOs
│   │   └── photo.go
│   └── database/                 # Database setup
│       ├── postgres.go
│       └── migrate.go
├── migrations/                   # SQL migration files
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── proto/                        # Protocol buffer definitions
│   └── photov1/photo.proto
├── api/                          # OpenAPI specifications
│   └── openapi.yaml
├── pkg/                          # Public packages (if any)
├── configs/                      # Configuration files
├── scripts/                      # Build and deployment scripts
├── test/                         # Integration tests
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── docker-compose.yaml
└── README.md
```

## Database Connection Pooling

```go
import "github.com/jackc/pgx/v5/pgxpool"

// Use pgxpool with connection pool configuration
config, err := pgxpool.ParseConfig(databaseURL)
if err != nil {
    return nil, fmt.Errorf("failed to parse database config: %w", err)
}

config.MaxConns = 50
config.MinConns = 10
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = time.Minute * 30

pool, err := pgxpool.NewWithConfig(context.Background(), config)
```

## API Key Authentication

```go
// Use middleware for API key validation
func (h *Handler) APIKeyAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        apiKey := r.Header.Get("X-API-Key")
        if apiKey == "" {
            http.Error(w, "missing API key", http.StatusUnauthorized)
            return
        }

        // Validate API key
        ctx, err := h.auth.ValidateAPIKey(r.Context(), apiKey)
        if err != nil {
            http.Error(w, "invalid API key", http.StatusUnauthorized)
            return
        }

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Environment Variables

Read configuration from environment variables using a library like `cleanenv` or `viper`:

```go
type Config struct {
    DatabaseURLstring `env:"DATABASE_URL" env-required:"true"`
    GCSBucketNamestring `env:"GCS_BUCKET_NAME" env-required:"true"`
    LRSServiceHost    string `env:"LRS_SERVICE_HOST" env-required:"true"`
    LRSServicePort    int`env:"LRS_SERVICE_PORT" env-default:"50051"`
    Portint `env:"PORT" env-default:"8080"`
    LogLevel         string `env:"LOG_LEVEL" env-default:"info"`
}
```

## Security

- Never log API keys, tokens, or sensitive data
- Use environment variables for secrets (not code)
- Validate all user inputs
- Use parameterized queries (pgx does this by default)
- Use TLS for all external communication
- Rate limit API endpoints (100 requests/min per API key)

## PRD Reference

Always refer to `PRD.md` for:
- Complete API specifications
- Two-phase upload architecture
- Error codes and statuses
- Database schema design
- Security requirements