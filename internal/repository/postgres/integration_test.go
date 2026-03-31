//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

const (
	defaultTestDBURL = "postgres://postgres:1234@localhost:5432/bm_photos_test?sslmode=disable"
)

type testDBConfig struct {
	dbURL string
}

func getTestDBConfig() *testDBConfig {
	return &testDBConfig{
		dbURL: getEnvOrDefault("TEST_DATABASE_URL", defaultTestDBURL),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupTestDB(ctx context.Context) (*PostgresDB, func(), error) {
	cfg := getTestDBConfig()

	poolConfig, err := pgxpool.ParseConfig(cfg.dbURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &PostgresDB{pool: pool}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup, nil
}

func runMigrations(ctx context.Context, db *PostgresDB) error {
	migration := `
	CREATE TABLE IF NOT EXISTS photos (
		photo_id UUID PRIMARY KEY,
		route_id VARCHAR(50) NOT NULL,
		lane_code VARCHAR(10) NOT NULL,
		latitude DOUBLE PRECISION NOT NULL,
		longitude DOUBLE PRECISION NOT NULL,
		sta_value DOUBLE PRECISION NOT NULL,
		sta_source VARCHAR(20) NOT NULL DEFAULT 'user_provided',
		gcs_object_name VARCHAR(500) NOT NULL,
		thumbnail_small_path VARCHAR(500),
		thumbnail_medium_path VARCHAR(500),
		thumbnail_large_path VARCHAR(500),
		file_format VARCHAR(10) NOT NULL,
		file_size_bytes BIGINT NOT NULL,
		original_filename VARCHAR(255),
		exif_data JSONB,
		description TEXT,
		tags JSONB DEFAULT '[]',
		upload_token UUID NOT NULL UNIQUE,
		upload_status VARCHAR(20) NOT NULL DEFAULT 'pending',
		uploaded_by VARCHAR(100) NOT NULL,
		uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'processing',
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		processing_completed_at TIMESTAMP WITH TIME ZONE,
		deleted_at TIMESTAMP WITH TIME ZONE,
		deleted_by VARCHAR(100)
	);

	CREATE TABLE IF NOT EXISTS pending_uploads (
		upload_token UUID PRIMARY KEY,
		photo_id UUID NOT NULL REFERENCES photos(photo_id),
		api_key_id VARCHAR(100) NOT NULL,
		file_name VARCHAR(255) NOT NULL,
		content_type VARCHAR(100) NOT NULL,
		file_size_bytes BIGINT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		completed_at TIMESTAMP WITH TIME ZONE,
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

	_, err := db.pool.Exec(ctx, migration)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func cleanupTables(ctx context.Context, db *PostgresDB) error {
	tables := []string{
		"photo_audit_log",
		"pending_uploads",
		"photos",
		"api_keys",
	}

	for _, table := range tables {
		_, err := db.pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			return fmt.Errorf("failed to cleanup table %s: %w", table, err)
		}
	}

	return nil
}

type PhotoBuilder struct {
	photo *entity.Photo
}

func NewPhotoBuilder() *PhotoBuilder {
	return &PhotoBuilder{}
}

func (b *PhotoBuilder) WithRouteID(routeID string) *PhotoBuilder {
	params := entity.PhotoParams{
		RouteID:       routeID,
		LaneCode:      "L1",
		FileFormat:    vo.FileFormatJPEG,
		FileSizeBytes: 1024,
		UploadToken:   vo.NewUploadToken(),
		UploadedBy:    "test-api-key",
	}
	photo, _ := entity.NewPhoto(params)
	photo.SetSTA(0, vo.STASourceUserProvided)
	b.photo = photo
	return b
}

func (b *PhotoBuilder) WithLaneCode(laneCode string) *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.SetLaneCode(laneCode)
	return b
}

func (b *PhotoBuilder) WithSTA(staValue float64, source vo.STASource) *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.SetSTA(staValue, source)
	return b
}

func (b *PhotoBuilder) WithCoordinates(lat, lon float64) *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.SetCoordinates(lat, lon)
	return b
}

func (b *PhotoBuilder) WithDescription(desc string) *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.UpdateDescription(desc)
	return b
}

func (b *PhotoBuilder) WithTags(tags []string) *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.UpdateTags(tags)
	return b
}

func (b *PhotoBuilder) WithEXIFData(exif *entity.EXIFData) *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.SetEXIFData(exif)
	return b
}

func (b *PhotoBuilder) MarkUploadComplete() *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.MarkUploadComplete()
	return b
}

func (b *PhotoBuilder) MarkProcessingComplete(thumbnails entity.ThumbnailPaths) *PhotoBuilder {
	if b.photo == nil {
		return b.WithRouteID("NR-001")
	}
	b.photo.MarkProcessingComplete(thumbnails)
	return b
}

func (b *PhotoBuilder) Build() *entity.Photo {
	if b.photo == nil {
		b.WithRouteID("NR-001")
	}
	return b.photo
}

func (b *PhotoBuilder) Create(ctx context.Context, repo *PhotoRepository) (*entity.Photo, error) {
	photo := b.Build()
	err := repo.Create(ctx, photo)
	if err != nil {
		return nil, err
	}
	return photo, nil
}

type PendingUploadBuilder struct {
	upload *PendingUpload
}

type PendingUpload = repository.PendingUpload

func NewPendingUploadBuilder() *PendingUploadBuilder {
	return &PendingUploadBuilder{}
}

func (b *PendingUploadBuilder) WithPhotoID(photoID vo.PhotoID) *PendingUploadBuilder {
	b.upload.PhotoID = photoID
	return b
}

func (b *PendingUploadBuilder) WithAPIKeyID(apiKeyID string) *PendingUploadBuilder {
	b.upload.APIKeyID = apiKeyID
	return b
}

func (b *PendingUploadBuilder) WithFilename(filename string) *PendingUploadBuilder {
	b.upload.Filename = filename
	return b
}

func (b *PendingUploadBuilder) WithStatus(status vo.UploadStatus) *PendingUploadBuilder {
	b.upload.Status = status
	return b
}

func (b *PendingUploadBuilder) WithExpiresIn(duration time.Duration) *PendingUploadBuilder {
	b.upload.ExpiresAt = time.Now().Add(duration)
	return b
}

func (b *PendingUploadBuilder) Build() *repository.PendingUpload {
	if b.upload == nil {
		b.upload = &repository.PendingUpload{
			UploadToken:   vo.NewUploadToken(),
			PhotoID:       vo.NewPhotoID(),
			APIKeyID:      "test-api-key",
			Filename:      "test.jpg",
			ContentType:   "image/jpeg",
			FileSizeBytes: 1024,
			CreatedAt:     time.Now(),
			ExpiresAt:     time.Now().Add(15 * time.Minute),
			Status:        vo.UploadStatusPending,
		}
	}
	return b.upload
}

func (b *PendingUploadBuilder) Create(ctx context.Context, repo *PendingUploadRepository) (*repository.PendingUpload, error) {
	upload := b.Build()
	err := repo.Create(ctx, upload)
	if err != nil {
		return nil, err
	}
	return upload, nil
}
