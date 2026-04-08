package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bina-marga/survey-photo/internal/service"
)

type contextKey string

const (
	APIKeyIDKey     contextKey = "api_key_id"
	APIKeyScopesKey contextKey = "api_key_scopes"
)

type Middleware struct {
	authService service.AuthService
	logger      *slog.Logger
}

func NewMiddleware(authService service.AuthService, logger *slog.Logger) *Middleware {
	return &Middleware{
		authService: authService,
		logger:      logger,
	}
}

func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		m.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (m *Middleware) APIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			http.Error(w, `{"error": "missing API key", "code": "MISSING_API_KEY"}`, http.StatusUnauthorized)
			return
		}

		apiKeyRecord, err := m.authService.ValidateAPIKey(r.Context(), apiKey)
		if err != nil {
			m.logger.Warn("invalid API key", "error", err)
			http.Error(w, `{"error": "invalid API key", "code": "INVALID_API_KEY"}`, http.StatusUnauthorized)
			return
		}

		if !apiKeyRecord.IsActive {
			http.Error(w, `{"error": "API key is inactive", "code": "INACTIVE_API_KEY"}`, http.StatusUnauthorized)
			return
		}

		if apiKeyRecord.ExpiresAt != nil && apiKeyRecord.ExpiresAt.Before(time.Now()) {
			http.Error(w, `{"error": "API key has expired", "code": "EXPIRED_API_KEY"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), APIKeyIDKey, apiKeyRecord.KeyID)
		ctx = context.WithValue(ctx, APIKeyScopesKey, apiKeyRecord.Scopes)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *Middleware) RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, ok := r.Context().Value(APIKeyScopesKey).([]string)
			if !ok {
				http.Error(w, `{"error": "forbidden", "code": "FORBIDDEN"}`, http.StatusForbidden)
				return
			}

			hasScope := false
			for _, s := range scopes {
				if s == scope {
					hasScope = true
					break
				}
			}

			if !hasScope {
				http.Error(w, `{"error": "insufficient scope", "code": "INSUFFICIENT_SCOPE"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.RemoteAddr
		}

		now := time.Now()
		windowStart := now.Add(-rl.window)

		requests := rl.requests[key]
		var validRequests []time.Time
		for _, t := range requests {
			if t.After(windowStart) {
				validRequests = append(validRequests, t)
			}
		}

		if len(validRequests) >= rl.limit {
			http.Error(w, `{"error": "rate limit exceeded", "code": "RATE_LIMIT_EXCEEDED"}`, http.StatusTooManyRequests)
			return
		}

		validRequests = append(validRequests, now)
		rl.requests[key] = validRequests

		next.ServeHTTP(w, r)
	})
}

func GetAPIKeyID(ctx context.Context) string {
	if id, ok := ctx.Value(APIKeyIDKey).(string); ok {
		return id
	}
	return ""
}

func GetAPIKeyScopes(ctx context.Context) []string {
	if scopes, ok := ctx.Value(APIKeyScopesKey).([]string); ok {
		return scopes
	}
	return nil
}

func HasScope(ctx context.Context, scope string) bool {
	scopes := GetAPIKeyScopes(ctx)
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func ParseQueryInt(r *http.Request, name string, defaultVal int) int {
	if val := r.URL.Query().Get(name); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func ParseQueryFloat64(r *http.Request, name string) *float64 {
	if val := r.URL.Query().Get(name); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return &f
		}
	}
	return nil
}

func ParseQueryString(r *http.Request, name string) *string {
	if val := r.URL.Query().Get(name); val != "" {
		return &val
	}
	return nil
}

func ParseQueryBool(r *http.Request, name string) *bool {
	if val := r.URL.Query().Get(name); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return &b
		}
	}
	return nil
}

func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func NormalizeContentType(ct string) string {
	ct = strings.ToLower(ct)
	if ct == "image/jpg" {
		return "image/jpeg"
	}
	return ct
}
