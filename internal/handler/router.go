package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bina-marga/survey-photo/internal/service"
)

// Rate limit configuration: 100 requests per minute per API key
const (
	rateLimitPerMinute = 100
	rateLimitWindow    = time.Minute
)

// NewRouter creates and configures the HTTP router with all routes and middleware
func NewRouter(
	uploadSvc service.UploadService,
	photoSvc service.PhotoService,
	authSvc service.AuthService,
	gcsClient service.GCSClient,
	dbPinger interface {
		Ping(ctx context.Context) error
	},
	logger *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	// Initialize handlers
	healthHandler := NewHealthHandler(dbPinger)
	uploadHandler := NewUploadHandler(uploadSvc, logger)
	photoHandler := NewPhotoHandler(photoSvc, gcsClient, logger)

	// Initialize middleware
	mw := NewMiddleware(authSvc, logger)
	rateLimiter := NewRateLimiter(rateLimitPerMinute, rateLimitWindow)

	// Health endpoints (no auth required)
	mux.Handle("GET /health", http.HandlerFunc(healthHandler.HealthCheck))
	mux.Handle("GET /ready", http.HandlerFunc(healthHandler.ReadinessCheck))

	// Upload endpoints (write scope required)
	mux.Handle("POST /api/v1/photos/upload-url", withMiddleware(
		http.HandlerFunc(uploadHandler.GetSignedUploadURL),
		mw.Logging, mw.CORS, rateLimiter.Middleware, mw.APIKeyAuth, mw.RequireScope("write"),
	))
	mux.Handle("POST /api/v1/photos/confirm", withMiddleware(
		http.HandlerFunc(uploadHandler.ConfirmUpload),
		mw.Logging, mw.CORS, rateLimiter.Middleware, mw.APIKeyAuth, mw.RequireScope("write"),
	))

	// Photo read endpoints (read scope required)
	mux.Handle("GET /api/v1/photos/{photo_id}", withMiddleware(
		http.HandlerFunc(photoHandler.GetPhoto),
		mw.Logging, mw.CORS, rateLimiter.Middleware, mw.APIKeyAuth, mw.RequireScope("read"),
	))
	mux.Handle("GET /api/v1/photos/{photo_id}/download", withMiddleware(
		http.HandlerFunc(photoHandler.DownloadPhoto),
		mw.Logging, mw.CORS, rateLimiter.Middleware, mw.APIKeyAuth, mw.RequireScope("read"),
	))
	mux.Handle("GET /api/v1/photos", withMiddleware(
		http.HandlerFunc(photoHandler.BrowsePhotos),
		mw.Logging, mw.CORS, rateLimiter.Middleware, mw.APIKeyAuth, mw.RequireScope("read"),
	))

	// Photo write endpoints (write scope required)
	mux.Handle("PATCH /api/v1/photos/{photo_id}", withMiddleware(
		http.HandlerFunc(photoHandler.UpdatePhoto),
		mw.Logging, mw.CORS, rateLimiter.Middleware, mw.APIKeyAuth, mw.RequireScope("write"),
	))

	// Photo admin endpoints (admin scope required)
	mux.Handle("DELETE /api/v1/photos/{photo_id}", withMiddleware(
		http.HandlerFunc(photoHandler.DeletePhoto),
		mw.Logging, mw.CORS, rateLimiter.Middleware, mw.APIKeyAuth, mw.RequireScope("admin"),
	))

	return mux
}

// withMiddleware applies middleware to a handler in reverse order
// so they execute in the specified order (first listed = first executed)
func withMiddleware(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
