//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

func TestPendingUploadRepository_CreateAndGetByToken(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	upload := &repository.PendingUpload{
		UploadToken: vo.NewUploadToken(),
		PhotoID:     photo.ID(),
		APIKeyID:    "test-api-key",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(15 * time.Minute),
		Status:      vo.UploadStatusPending,
	}

	err = uploadRepo.Create(ctx, upload)
	require.NoError(t, err)

	retrieved, err := uploadRepo.GetByToken(ctx, upload.UploadToken)
	require.NoError(t, err)
	assert.Equal(t, upload.PhotoID, retrieved.PhotoID)
	assert.Equal(t, vo.UploadStatusPending, retrieved.Status)
}

func TestPendingUploadRepository_GetByPhotoID(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	upload := &repository.PendingUpload{
		UploadToken: vo.NewUploadToken(),
		PhotoID:     photo.ID(),
		APIKeyID:    "test-api-key",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(15 * time.Minute),
		Status:      vo.UploadStatusPending,
	}

	err = uploadRepo.Create(ctx, upload)
	require.NoError(t, err)

	retrieved, err := uploadRepo.GetByPhotoID(ctx, photo.ID())
	require.NoError(t, err)
	assert.Equal(t, upload.UploadToken, retrieved.UploadToken)
}

func TestPendingUploadRepository_GetByToken_NotFound(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	uploadRepo := NewPendingUploadRepository(db)

	nonExistentToken := vo.NewUploadToken()
	_, err = uploadRepo.GetByToken(ctx, nonExistentToken)
	assert.ErrorIs(t, err, repository.ErrTokenNotFound)
}

func TestPendingUploadRepository_MarkAsCompleted(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	upload := &repository.PendingUpload{
		UploadToken: vo.NewUploadToken(),
		PhotoID:     photo.ID(),
		APIKeyID:    "test-api-key",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(15 * time.Minute),
		Status:      vo.UploadStatusPending,
	}

	err = uploadRepo.Create(ctx, upload)
	require.NoError(t, err)

	err = uploadRepo.MarkAsCompleted(ctx, upload.UploadToken)
	require.NoError(t, err)

	retrieved, err := uploadRepo.GetByToken(ctx, upload.UploadToken)
	require.NoError(t, err)
	assert.Equal(t, vo.UploadStatusCompleted, retrieved.Status)
}

func TestPendingUploadRepository_MarkAsExpired(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	upload := &repository.PendingUpload{
		UploadToken: vo.NewUploadToken(),
		PhotoID:     photo.ID(),
		APIKeyID:    "test-api-key",
		CreatedAt:   time.Now().Add(-30 * time.Minute),
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
		Status:      vo.UploadStatusPending,
	}

	err = uploadRepo.Create(ctx, upload)
	require.NoError(t, err)

	count, err := uploadRepo.MarkAsExpired(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	retrieved, err := uploadRepo.GetByToken(ctx, upload.UploadToken)
	require.NoError(t, err)
	assert.Equal(t, vo.UploadStatusExpired, retrieved.Status)
}

func TestPendingUploadRepository_Delete(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	upload := &repository.PendingUpload{
		UploadToken: vo.NewUploadToken(),
		PhotoID:     photo.ID(),
		APIKeyID:    "test-api-key",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(15 * time.Minute),
		Status:      vo.UploadStatusPending,
	}

	err = uploadRepo.Create(ctx, upload)
	require.NoError(t, err)

	err = uploadRepo.Delete(ctx, upload.UploadToken)
	require.NoError(t, err)

	_, err = uploadRepo.GetByToken(ctx, upload.UploadToken)
	assert.ErrorIs(t, err, repository.ErrTokenNotFound)
}

func TestPendingUploadRepository_DeleteExpired(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		upload := &repository.PendingUpload{
			UploadToken: vo.NewUploadToken(),
			PhotoID:     photo.ID(),
			APIKeyID:    "test-api-key",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			ExpiresAt:   time.Now().Add(-1 * time.Hour),
			Status:      vo.UploadStatusExpired,
		}
		err = uploadRepo.Create(ctx, upload)
		require.NoError(t, err)
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	count, err := uploadRepo.DeleteExpired(ctx, oneHourAgo)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestPendingUploadRepository_CountActiveByAPIKey(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	apiKeyID := "test-api-key-1"

	for i := 0; i < 3; i++ {
		upload := &repository.PendingUpload{
			UploadToken: vo.NewUploadToken(),
			PhotoID:     photo.ID(),
			APIKeyID:    apiKeyID,
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(15 * time.Minute),
			Status:      vo.UploadStatusPending,
		}
		err = uploadRepo.Create(ctx, upload)
		require.NoError(t, err)
	}

	count, err := uploadRepo.CountActiveByAPIKey(ctx, apiKeyID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	otherAPIKeyCount, err := uploadRepo.CountActiveByAPIKey(ctx, "other-api-key")
	require.NoError(t, err)
	assert.Equal(t, int64(0), otherAPIKeyCount)
}

func TestPendingUploadRepository_GetExpired(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	photoRepo := NewPhotoRepository(db)
	uploadRepo := NewPendingUploadRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()
	err = photoRepo.Create(ctx, photo)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		upload := &repository.PendingUpload{
			UploadToken: vo.NewUploadToken(),
			PhotoID:     photo.ID(),
			APIKeyID:    "test-api-key",
			CreatedAt:   time.Now().Add(-30 * time.Minute),
			ExpiresAt:   time.Now().Add(-1 * time.Hour),
			Status:      vo.UploadStatusPending,
		}
		err = uploadRepo.Create(ctx, upload)
		require.NoError(t, err)
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	expired, err := uploadRepo.GetExpired(ctx, oneHourAgo)
	require.NoError(t, err)
	assert.Len(t, expired, 3)
}
