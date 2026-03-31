//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

func TestPhotoRepository_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		WithLaneCode("L1").
		WithSTA(10.5, vo.STASourceUserProvided).
		WithCoordinates(-6.2088, 106.8456).
		Build()

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, photo.ID())
	require.NoError(t, err)
	assert.Equal(t, photo.RouteID(), retrieved.RouteID())
	assert.Equal(t, photo.LaneCode(), retrieved.LaneCode())
	assert.Equal(t, photo.STAValue(), retrieved.STAValue())
	assert.Equal(t, photo.STAValue(), 10.5)
}

func TestPhotoRepository_Create_GetByUploadToken(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-002").
		WithLaneCode("R1").
		Build()

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	retrieved, err := repo.GetByUploadToken(ctx, photo.UploadToken())
	require.NoError(t, err)
	assert.Equal(t, photo.ID(), retrieved.ID())
	assert.Equal(t, photo.RouteID(), "NR-002")
}

func TestPhotoRepository_GetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	nonExistentID := vo.NewPhotoID()
	_, err = repo.GetByID(ctx, nonExistentID)
	assert.ErrorIs(t, err, repository.ErrPhotoNotFound)
}

func TestPhotoRepository_Update(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	photo.UpdateDescription("Updated description")
	photo.UpdateTags([]string{"tag1", "tag2"})

	err = repo.Update(ctx, photo)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, photo.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated description", *retrieved.Description())
	assert.Equal(t, []string{"tag1", "tag2"}, retrieved.Tags())
}

func TestPhotoRepository_SoftDelete(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	err = repo.SoftDelete(ctx, photo.ID(), "admin-api-key")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, photo.ID())
	assert.ErrorIs(t, err, repository.ErrPhotoNotFound)

	exists, err := repo.Exists(ctx, photo.ID())
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPhotoRepository_Restore(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	err = repo.SoftDelete(ctx, photo.ID(), "admin-api-key")
	require.NoError(t, err)

	err = repo.Restore(ctx, photo.ID())
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, photo.ID())
	require.NoError(t, err)
	assert.Equal(t, photo.ID(), retrieved.ID())
	assert.False(t, retrieved.IsDeleted())
}

func TestPhotoRepository_HardDelete(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	err = repo.HardDelete(ctx, photo.ID())
	require.NoError(t, err)

	exists, err := repo.Exists(ctx, photo.ID())
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestPhotoRepository_UpdateSTA(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		WithSTA(5.0, vo.STASourceUserProvided).
		Build()

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	newSTA := 15.5
	newSource := vo.STASourceLRSInterpolated
	err = repo.UpdateSTA(ctx, photo.ID(), &newSTA, &newSource)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, photo.ID())
	require.NoError(t, err)
	assert.NotNil(t, retrieved.STAValue())
	assert.Equal(t, 15.5, *retrieved.STAValue())
	assert.NotNil(t, retrieved.STASource())
	assert.Equal(t, vo.STASourceLRSInterpolated, *retrieved.STASource())
}

func TestPhotoRepository_Browse_Basic(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	for i := 0; i < 5; i++ {
		photo := NewPhotoBuilder().
			WithRouteID("NR-001").
			WithSTA(float64(i)*10, vo.STASourceUserProvided).
			Build()
		err = repo.Create(ctx, photo)
		require.NoError(t, err)
	}

	result, err := repo.Browse(ctx, repository.BrowseFilter{
		RouteID: "NR-001",
		Page:    1,
		PerPage: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.TotalCount)
	assert.Len(t, result.Photos, 5)
}

func TestPhotoRepository_Browse_Pagination(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	for i := 0; i < 15; i++ {
		photo := NewPhotoBuilder().
			WithRouteID("NR-001").
			Build()
		err = repo.Create(ctx, photo)
		require.NoError(t, err)
	}

	result, err := repo.Browse(ctx, repository.BrowseFilter{
		Page:    2,
		PerPage: 5,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(15), result.TotalCount)
	assert.Len(t, result.Photos, 5)
	assert.Equal(t, 2, result.Page)
	assert.Equal(t, 5, result.PerPage)
}

func TestPhotoRepository_Browse_STAFilter(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	for i := 0; i < 10; i++ {
		photo := NewPhotoBuilder().
			WithRouteID("NR-001").
			WithSTA(float64(i)*10, vo.STASourceUserProvided).
			Build()
		err = repo.Create(ctx, photo)
		require.NoError(t, err)
	}

	staStart := 20.0
	staEnd := 50.0

	result, err := repo.Browse(ctx, repository.BrowseFilter{
		RouteID:  "NR-001",
		STAStart: &staStart,
		STAEnd:   &staEnd,
		Page:     1,
		PerPage:  100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), result.TotalCount)
}

func TestPhotoRepository_Browse_LaneFilter(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	lanes := []string{"L1", "L2", "R1"}
	for _, lane := range lanes {
		for j := 0; j < 3; j++ {
			photo := NewPhotoBuilder().
				WithRouteID("NR-001").
				WithLaneCode(lane).
				WithSTA(float64(j), vo.STASourceUserProvided).
				Build()
			err = repo.Create(ctx, photo)
			require.NoError(t, err)
		}
	}

	laneFilter := "L1"
	result, err := repo.Browse(ctx, repository.BrowseFilter{
		RouteID: "NR-001",
		Lane:    &laneFilter,
		Page:    1,
		PerPage: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.TotalCount)
}

func TestPhotoRepository_Search_RouteIDs(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	routes := []string{"NR-001", "NR-002", "NR-003"}
	for _, route := range routes {
		for i := 0; i < 2; i++ {
			photo := NewPhotoBuilder().
				WithRouteID(route).
				Build()
			err = repo.Create(ctx, photo)
			require.NoError(t, err)
		}
	}

	result, err := repo.Search(ctx, repository.SearchFilter{
		RouteIDs: []string{"NR-001", "NR-002"},
		Page:     1,
		PerPage:  100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), result.TotalCount)
}

func TestPhotoRepository_Search_STARanges(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	for i := 0; i < 10; i++ {
		photo := NewPhotoBuilder().
			WithRouteID("NR-001").
			WithSTA(float64(i)*10, vo.STASourceUserProvided).
			Build()
		err = repo.Create(ctx, photo)
		require.NoError(t, err)
	}

	result, err := repo.Search(ctx, repository.SearchFilter{
		STARanges: []repository.STARange{
			{Start: 20.0, End: 50.0},
		},
		Page:    1,
		PerPage: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(4), result.TotalCount)
}

func TestPhotoRepository_Search_DateRange(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	for i := 0; i < 5; i++ {
		photo := NewPhotoBuilder().
			WithRouteID("NR-001").
			Build()
		err = repo.Create(ctx, photo)
		require.NoError(t, err)
	}

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	result, err := repo.Search(ctx, repository.SearchFilter{
		DateStart: &yesterday,
		DateEnd:   &tomorrow,
		Page:      1,
		PerPage:   100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.TotalCount)
}

func TestPhotoRepository_Search_TagsFilter(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	tags := [][]string{
		{"tag1", "tag2"},
		{"tag2", "tag3"},
		{"tag3", "tag4"},
		{"tag1"},
	}

	for _, photoTags := range tags {
		_ = photoTags // avoid unused variable warning
		photo := NewPhotoBuilder().
			WithRouteID("NR-001").
			WithTags(photoTags).
			Build()
		err = repo.Create(ctx, photo)
		require.NoError(t, err)
	}

	result, err := repo.Search(ctx, repository.SearchFilter{
		Tags:    []string{"tag1", "tag2"},
		Page:    1,
		PerPage: 100,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.TotalCount)
}

func TestPhotoRepository_Exists(t *testing.T) {
	ctx := context.Background()
	db, cleanup, err := setupTestDB(ctx)
	require.NoError(t, err)
	defer cleanup()

	err = runMigrations(ctx, db)
	require.NoError(t, err)

	err = cleanupTables(ctx, db)
	require.NoError(t, err)

	repo := NewPhotoRepository(db)

	photo := NewPhotoBuilder().
		WithRouteID("NR-001").
		Build()

	exists, err := repo.Exists(ctx, photo.ID())
	require.NoError(t, err)
	assert.False(t, exists)

	err = repo.Create(ctx, photo)
	require.NoError(t, err)

	exists, err = repo.Exists(ctx, photo.ID())
	require.NoError(t, err)
	assert.True(t, exists)
}

func strPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
