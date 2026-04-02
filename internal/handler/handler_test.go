//go:build integration

package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bina-marga/survey-photo/internal/client/gcs"
	"github.com/bina-marga/survey-photo/internal/config"
	"github.com/bina-marga/survey-photo/internal/repository"
	"github.com/bina-marga/survey-photo/internal/repository/postgres"
	"github.com/bina-marga/survey-photo/internal/service"
	"github.com/joho/godotenv"
)

// slogLogger wraps slog.Logger to implement service.Logger interface
type slogLogger struct {
	*slog.Logger
}

// Compile-time interface check
var _ service.Logger = (*slogLogger)(nil)

func (l *slogLogger) Info(msg string, ctx ...map[string]interface{}) {
	if len(ctx) > 0 {
		args := make([]any, 0, len(ctx[0])*2)
		for k, v := range ctx[0] {
			args = append(args, k, v)
		}
		l.Logger.Info(msg, args...)
	} else {
		l.Logger.Info(msg)
	}
}

func (l *slogLogger) Error(msg string, err error, ctx ...map[string]interface{}) {
	args := []any{"error", err}
	if len(ctx) > 0 {
		for k, v := range ctx[0] {
			args = append(args, k, v)
		}
	}
	l.Logger.Error(msg, args...)
}

func (l *slogLogger) Warn(msg string, ctx ...map[string]interface{}) {
	if len(ctx) > 0 {
		args := make([]any, 0, len(ctx[0])*2)
		for k, v := range ctx[0] {
			args = append(args, k, v)
		}
		l.Logger.Warn(msg, args...)
	} else {
		l.Logger.Warn(msg)
	}
}

func (l *slogLogger) Debug(msg string, ctx ...map[string]interface{}) {
	if len(ctx) > 0 {
		args := make([]any, 0, len(ctx[0])*2)
		for k, v := range ctx[0] {
			args = append(args, k, v)
		}
		l.Logger.Debug(msg, args...)
	} else {
		l.Logger.Debug(msg)
	}
}

// testServer holds the test server and its dependencies
type testServer struct {
	*httptest.Server
	photoRepo         repository.PhotoRepository
	pendingUploadRepo repository.PendingUploadRepository
	apiKeyRepo        repository.APIKeyRepository
	auditLogRepo      repository.AuditLogRepository
	gcsClient         service.GCSClient
	db                *postgres.PostgresDB
}

// setupTestServer creates a fully configured test server with all dependencies
func setupTestServer(t *testing.T) *testServer {
	t.Helper()

	ctx := context.Background()

	// Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping integration test: failed to load config: %v", err)
	}

	// Build database connection URL
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	// Connect to PostgreSQL
	db, err := postgres.NewPostgresDB(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: failed to connect to database: %v", err)
	}

	// Run migrations
	if err := runTestMigrations(ctx, db); err != nil {
		db.Close()
		t.Skipf("Skipping integration test: failed to run migrations: %v", err)
	}

	// Clean up tables before test
	if err := cleanupTestTables(ctx, db); err != nil {
		db.Close()
		t.Skipf("Skipping integration test: failed to cleanup tables: %v", err)
	}

	// Initialize GCS client
	gcsConfig := gcs.Config{
		BucketName:          cfg.GCS.BucketName,
		CredentialsPath:     cfg.GCS.CredentialsPath,
		TestPrefix:          cfg.GCS.TestPrefix,
		SignedURLExpiryMins: 15,
		ConnectTimeoutSecs:  30,
	}

	gcsClient, err := gcs.NewClient(ctx, gcsConfig)
	if err != nil {
		db.Close()
		t.Skipf("Skipping integration test: failed to create GCS client: %v", err)
	}

	// Initialize repositories
	photoRepo := postgres.NewPhotoRepository(db)
	pendingUploadRepo := postgres.NewPendingUploadRepository(db)
	apiKeyRepo := postgres.NewAPIKeyRepository(db)
	auditLogRepo := postgres.NewAuditLogRepository(db)

	// Initialize services with wrapped logger
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	wrappedLogger := &slogLogger{Logger: logger}
	auditSvc := service.NewAuditLogService(auditLogRepo, wrappedLogger)
	photoSvc := service.NewPhotoService(photoRepo, pendingUploadRepo, gcsClient, wrappedLogger, auditSvc)
	uploadSvc := service.NewUploadService(photoRepo, pendingUploadRepo, gcsClient, wrappedLogger)
	authSvc := service.NewAuthService(apiKeyRepo, wrappedLogger)
	adminSvc := service.NewAdminService(apiKeyRepo, wrappedLogger)

	// Build router (uses slog.Logger directly)
	router := NewRouter(uploadSvc, photoSvc, authSvc, adminSvc, gcsClient, db, logger)

	// Create test server
	server := httptest.NewServer(router)

	ts := &testServer{
		Server:            server,
		photoRepo:         photoRepo,
		pendingUploadRepo: pendingUploadRepo,
		apiKeyRepo:        apiKeyRepo,
		auditLogRepo:      auditLogRepo,
		gcsClient:         gcsClient,
		db:                db,
	}

	// Register cleanup
	t.Cleanup(func() {
		// Clean up GCS objects created during tests
		// Note: individual tests should call cleanupGCSObject for specific objects

		// Close test server
		server.Close()

		// Close GCS client
		if gcsClient != nil {
			gcsClient.Close()
		}

		// Close database connection
		db.Close()
	})

	return ts
}

// runTestMigrations runs the necessary migrations for testing
func runTestMigrations(ctx context.Context, db *postgres.PostgresDB) error {
	migration := `
	CREATE TABLE IF NOT EXISTS photos (
		photo_id UUID PRIMARY KEY,
		route_id VARCHAR(50) NOT NULL,
		lane_code VARCHAR(10) NOT NULL,
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		sta_value DOUBLE PRECISION,
		sta_source VARCHAR(20),
		gcs_object_name VARCHAR(500) NOT NULL,
		file_format VARCHAR(10) NOT NULL,
		file_size_bytes BIGINT NOT NULL,
		original_filename VARCHAR(255),
		description TEXT,
		tags JSONB DEFAULT '[]',
		upload_token UUID NOT NULL UNIQUE,
		upload_status VARCHAR(20) NOT NULL DEFAULT 'pending',
		retry_count INTEGER NOT NULL DEFAULT 0,
		uploaded_by VARCHAR(100) NOT NULL,
		uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		deleted_at TIMESTAMP WITH TIME ZONE,
		deleted_by VARCHAR(100)
	);

	CREATE TABLE IF NOT EXISTS pending_uploads (
		upload_token UUID PRIMARY KEY,
		photo_id UUID NOT NULL REFERENCES photos(photo_id),
		api_key_id VARCHAR(100) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'pending'
	);

	CREATE TABLE IF NOT EXISTS api_keys (
		key_id UUID PRIMARY KEY,
		key_hash VARCHAR(255) NOT NULL UNIQUE,
		scopes JSONB NOT NULL DEFAULT '["read"]',
		description TEXT,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP WITH TIME ZONE,
		last_used_at TIMESTAMP WITH TIME ZONE,
		is_active BOOLEAN NOT NULL DEFAULT true
	);

	CREATE TABLE IF NOT EXISTS photo_audit_log (
		log_id UUID PRIMARY KEY,
		photo_id UUID REFERENCES photos(photo_id),
		operation VARCHAR(50) NOT NULL,
		api_key_id VARCHAR(100) NOT NULL,
		operated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		details JSONB
	);
	`

	_, err := db.Pool().Exec(ctx, migration)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Simplify pending_uploads table if needed
	simplifyPendingUploads := `
	ALTER TABLE pending_uploads DROP COLUMN IF EXISTS file_name;
	ALTER TABLE pending_uploads DROP COLUMN IF EXISTS content_type;
	ALTER TABLE pending_uploads DROP COLUMN IF EXISTS file_size_bytes;
	ALTER TABLE pending_uploads DROP COLUMN IF EXISTS completed_at;
	`
	_, err = db.Pool().Exec(ctx, simplifyPendingUploads)
	if err != nil {
		return fmt.Errorf("failed to simplify pending_uploads table: %w", err)
	}

	// Add retry_count column if it doesn't exist
	alterTable := `
	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'photos' AND column_name = 'retry_count'
		) THEN
			ALTER TABLE photos ADD COLUMN retry_count INTEGER NOT NULL DEFAULT 0;
		END IF;
	END $$;
	`
	_, err = db.Pool().Exec(ctx, alterTable)
	if err != nil {
		return fmt.Errorf("failed to add retry_count column: %w", err)
	}

	return nil
}

// cleanupTestTables truncates all test tables
func cleanupTestTables(ctx context.Context, db *postgres.PostgresDB) error {
	tables := []string{
		"photo_audit_log",
		"pending_uploads",
		"photos",
		"api_keys",
	}

	for _, table := range tables {
		_, err := db.Pool().Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to cleanup table %s: %w", table, err)
		}
	}

	return nil
}

// createTestAPIKey creates a test API key with the specified scopes
// Returns the API key record for use in tests
func createTestAPIKey(t *testing.T, repo repository.APIKeyRepository, scopes []string) *repository.APIKey {
	t.Helper()

	ctx := context.Background()

	// Generate random API key
	rawKey := generateRandomAPIKey()

	// Hash the key (SHA-256)
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Generate key ID for cleanup
	keyID := generateRandomKeyID()

	// Create API key record
	apiKey := &repository.APIKey{
		KeyID:       keyID,
		KeyHash:     keyHash,
		Scopes:      scopes,
		Description: "Test API key",
		CreatedAt:   time.Now(),
		IsActive:    true,
	}

	if err := repo.Create(ctx, apiKey); err != nil {
		t.Fatalf("Failed to create test API key: %v", err)
	}

	// Store raw key for use in tests (not persisted to DB)
	apiKey.RawKey = rawKey

	// Register cleanup to delete the API key after test
	t.Cleanup(func() {
		ctx := context.Background()
		_ = repo.Delete(ctx, apiKey.KeyID)
	})

	return apiKey
}

// generateRandomAPIKey generates a random API key string
func generateRandomAPIKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random API key: %v", err))
	}
	return hex.EncodeToString(bytes)
}

// generateRandomKeyID generates a random key ID
func generateRandomKeyID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random key ID: %v", err))
	}
	return hex.EncodeToString(bytes)
}

// doRequest makes an HTTP request to the test server with optional API key
func doRequest(t *testing.T, server *httptest.Server, method, path string, body io.Reader, apiKey *repository.APIKey) *http.Response {
	t.Helper()

	url := server.URL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != nil && apiKey.RawKey != "" {
		req.Header.Set("X-API-Key", apiKey.RawKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	return resp
}

// cleanupGCSObject deletes a GCS object after a test
func cleanupGCSObject(t *testing.T, gcsClient service.GCSClient, objectName string) {
	t.Helper()

	if objectName == "" {
		return
	}

	if err := gcsClient.DeleteFile(objectName); err != nil {
		t.Logf("Warning: failed to cleanup GCS object %s: %v", objectName, err)
	}
}

// parseJSONResponse parses a JSON response body into the given struct
func parseJSONResponse(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, string(body))
	}
}

// readResponseBody reads and returns the response body
func readResponseBody(t *testing.T, resp *http.Response) string {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	return string(body)
}

// TestMain is the entry point for all integration tests
func TestMain(m *testing.M) {
	// Load .env file if it exists
	envPath, err := filepath.Abs(filepath.Join("..", "..", ".env"))
	if err == nil {
		_ = godotenv.Load(envPath)
	}

	// Check if integration tests should run
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		fmt.Println("Skipping integration tests. Set RUN_INTEGRATION_TESTS=true to run.")
		os.Exit(0)
	}

	// Verify required environment variables are set
	required := []string{
		"DB_HOST",
		"DB_PORT",
		"DB_NAME",
		"DB_USERNAME",
		"DB_PASSWORD",
		"GCS_BUCKET_NAME",
		"GOOGLE_APPLICATION_CREDENTIALS",
	}

	var missing []string
	for _, env := range required {
		if os.Getenv(env) == "" {
			missing = append(missing, env)
		}
	}

	if len(missing) > 0 {
		fmt.Printf("Skipping integration tests: missing required environment variables: %v\n", missing)
		os.Exit(0)
	}

	os.Exit(m.Run())
}
