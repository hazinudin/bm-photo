// Package main provides the application entry point for the survey photo service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bina-marga/survey-photo/internal/client/gcs"
	"github.com/bina-marga/survey-photo/internal/config"
	"github.com/bina-marga/survey-photo/internal/handler"
	"github.com/bina-marga/survey-photo/internal/repository/postgres"
	"github.com/bina-marga/survey-photo/internal/service"
)

func main() {
	ctx := context.Background()

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize structured logger
	logLevel := parseLogLevel(cfg.Server.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting survey photo service", slog.Int("port", cfg.Server.Port))

	// 3. Connect to PostgreSQL
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	db, err := postgres.NewPostgresDB(ctx, connString)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	logger.Info("Connected to PostgreSQL database")

	// 4. Run database migrations (skip if migrations package doesn't exist)
	if err := runMigrations(ctx, db, logger); err != nil {
		logger.Warn("Database migrations skipped or failed", "error", err)
	}

	// 5. Initialize GCS client
	gcsClient, err := gcs.NewClient(ctx, gcs.Config{
		BucketName:      cfg.GCS.BucketName,
		CredentialsPath: cfg.GCS.CredentialsPath,
		TestPrefix:      cfg.GCS.TestPrefix,
	})
	if err != nil {
		logger.Error("Failed to initialize GCS client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := gcsClient.Close(); err != nil {
			logger.Warn("Failed to close GCS client", "error", err)
		}
	}()

	logger.Info("GCS client initialized", "bucket", cfg.GCS.BucketName)

	// 6. Initialize repositories
	photoRepo := postgres.NewPhotoRepository(db)
	pendingUploadRepo := postgres.NewPendingUploadRepository(db)
	apiKeyRepo := postgres.NewAPIKeyRepository(db)
	auditLogRepo := postgres.NewAuditLogRepository(db)

	logger.Info("Repositories initialized")

	// 7. Initialize services
	// Create a logger adapter for service.Logger interface
	serviceLogger := &slogLoggerAdapter{logger: logger}

	auditSvc := service.NewAuditLogService(auditLogRepo, serviceLogger)
	uploadSvc := service.NewUploadService(photoRepo, pendingUploadRepo, gcsClient, serviceLogger)
	photoSvc := service.NewPhotoService(photoRepo, gcsClient, serviceLogger, auditSvc)
	authSvc := service.NewAuthService(apiKeyRepo, serviceLogger)

	logger.Info("Services initialized")

	// 8. Build router
	router := handler.NewRouter(
		uploadSvc,
		photoSvc,
		authSvc,
		gcsClient,
		db,
		logger,
	)

	// 9. Start HTTP server with graceful shutdown
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("HTTP server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited gracefully")
}

// runMigrations attempts to run database migrations.
// If the migrations package doesn't exist, it logs a warning and returns nil.
func runMigrations(ctx context.Context, db *postgres.PostgresDB, logger *slog.Logger) error {
	// Try to import and run migrations if the package exists
	// For now, we'll just log that migrations weren't run since the package doesn't exist
	logger.Info("Database migrations not available - skipping")
	return nil
	// Note: When migrations package is added, uncomment and implement:
	// return database.Migrate(ctx, db.Pool())
}

// parseLogLevel converts a string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// slogLoggerAdapter adapts slog.Logger to service.Logger interface
type slogLoggerAdapter struct {
	logger *slog.Logger
}

// Info logs an informational message
func (a *slogLoggerAdapter) Info(msg string, ctx ...map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(ctx))
	for _, m := range ctx {
		for k, v := range m {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	a.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, attrs...)
}

// Error logs an error
func (a *slogLoggerAdapter) Error(msg string, err error, ctx ...map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(ctx)+1)
	attrs = append(attrs, slog.Any("error", err))
	for _, m := range ctx {
		for k, v := range m {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	a.logger.LogAttrs(context.Background(), slog.LevelError, msg, attrs...)
}

// Warn logs a warning
func (a *slogLoggerAdapter) Warn(msg string, ctx ...map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(ctx))
	for _, m := range ctx {
		for k, v := range m {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	a.logger.LogAttrs(context.Background(), slog.LevelWarn, msg, attrs...)
}

// Debug logs a debug message
func (a *slogLoggerAdapter) Debug(msg string, ctx ...map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(ctx))
	for _, m := range ctx {
		for k, v := range m {
			attrs = append(attrs, slog.Any(k, v))
		}
	}
	a.logger.LogAttrs(context.Background(), slog.LevelDebug, msg, attrs...)
}
