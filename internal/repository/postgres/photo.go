package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/bina-marga/survey-photo/internal/model/entity"
	"github.com/bina-marga/survey-photo/internal/model/vo"
	"github.com/bina-marga/survey-photo/internal/repository"
)

// PhotoRepository implements repository.PhotoRepository
type PhotoRepository struct {
	db *PostgresDB
}

// NewPhotoRepository creates a new PhotoRepository
func NewPhotoRepository(db *PostgresDB) *PhotoRepository {
	return &PhotoRepository{db: db}
}

// photoRow mirrors the DB schema for scanning
type photoRow struct {
	PhotoID          string
	RouteID          string
	LaneCode         string
	Latitude         *float64 // nullable
	Longitude        *float64 // nullable
	StaValue         *float64 // nullable
	StaSource        *string  // nullable
	GCSObjectName    string
	FileFormat       string
	FileSizeBytes    int64
	OriginalFilename *string
	Description      *string
	Tags             []string
	UploadToken      string
	UploadStatus     string
	RetryCount       int
	UploadedBy       string
	UploadedAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	DeletedBy        *string
}

func (r *photoRow) toEntity() (*entity.Photo, error) {
	// Parse enums
	photoID, err := vo.ParsePhotoID(r.PhotoID)
	if err != nil {
		return nil, fmt.Errorf("invalid photo ID: %w", err)
	}

	uploadToken, err := vo.ParseUploadToken(r.UploadToken)
	if err != nil {
		return nil, fmt.Errorf("invalid upload token: %w", err)
	}

	fileFormat, err := vo.ParseFileFormat(r.FileFormat)
	if err != nil {
		return nil, fmt.Errorf("invalid file format: %w", err)
	}

	uploadStatus, err := vo.ParseUploadStatus(r.UploadStatus)
	if err != nil {
		return nil, fmt.Errorf("invalid upload status: %w", err)
	}

	var staSource *vo.STASource
	if r.StaSource != nil {
		source, err := vo.ParseSTASource(*r.StaSource)
		if err != nil {
			return nil, fmt.Errorf("invalid STA source: %w", err)
		}
		staSource = &source
	}

	params := entity.PhotoRowParams{
		ID:               photoID,
		RouteID:          r.RouteID,
		LaneCode:         r.LaneCode,
		Latitude:         r.Latitude,
		Longitude:        r.Longitude,
		StaValue:         r.StaValue,
		StaSource:        staSource,
		GCSObjectName:    r.GCSObjectName,
		FileFormat:       fileFormat,
		FileSizeBytes:    r.FileSizeBytes,
		OriginalFilename: r.OriginalFilename,
		Description:      r.Description,
		Tags:             r.Tags,
		UploadToken:      uploadToken,
		UploadStatus:     uploadStatus,
		RetryCount:       r.RetryCount,
		UploadedBy:       r.UploadedBy,
		UploadedAt:       r.UploadedAt,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		DeletedAt:        r.DeletedAt,
		DeletedBy:        r.DeletedBy,
	}

	return entity.NewPhotoFromRepository(params), nil
}

func (r *PhotoRepository) Create(ctx context.Context, photo *entity.Photo) error {
	query := `
		INSERT INTO photos (
			photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, gcs_object_name,
			file_format, file_size_bytes, original_filename,
			description, tags,
			upload_token, upload_status, uploaded_by, uploaded_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13,
			$14, $15, $16, $17,
			$18, $19
		)`

	var staSourceStr *string
	if photo.STASource() != nil {
		str := photo.STASource().String()
		staSourceStr = &str
	}

	_, err := r.db.Pool().Exec(ctx, query,
		photo.ID().String(),
		photo.RouteID(),
		photo.LaneCode(),
		photo.Latitude(),
		photo.Longitude(),
		photo.STAValue(),
		staSourceStr,
		photo.GCSObjectName(),
		photo.FileFormat().String(),
		photo.FileSizeBytes(),
		photo.OriginalFilename(),
		photo.Description(),
		photo.Tags(),
		photo.UploadToken().String(),
		photo.UploadStatus().String(),
		photo.UploadedBy(),
		photo.UploadedAt(),
		photo.CreatedAt(),
		photo.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to create photo: %w", err)
	}

	return nil
}

func (r *PhotoRepository) GetByID(ctx context.Context, id vo.PhotoID) (*entity.Photo, error) {
	query := `
		SELECT photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, gcs_object_name,
			file_format, file_size_bytes, original_filename,
			description, tags,
			upload_token, upload_status, retry_count, uploaded_by, uploaded_at,
			created_at, updated_at,
			deleted_at, deleted_by
		FROM photos
		WHERE photo_id = $1 AND deleted_at IS NULL`

	row := r.db.Pool().QueryRow(ctx, query, id.String())

	return r.scanPhoto(row)
}

func (r *PhotoRepository) GetByIDIncludeDeleted(ctx context.Context, id vo.PhotoID) (*entity.Photo, error) {
	query := `
		SELECT photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, gcs_object_name,
			file_format, file_size_bytes, original_filename,
			description, tags,
			upload_token, upload_status, retry_count, uploaded_by, uploaded_at,
			created_at, updated_at,
			deleted_at, deleted_by
		FROM photos
		WHERE photo_id = $1`

	row := r.db.Pool().QueryRow(ctx, query, id.String())

	return r.scanPhoto(row)
}

func (r *PhotoRepository) GetByUploadToken(ctx context.Context, token vo.UploadToken) (*entity.Photo, error) {
	query := `
		SELECT photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, gcs_object_name,
			file_format, file_size_bytes, original_filename,
			description, tags,
			upload_token, upload_status, retry_count, uploaded_by, uploaded_at,
			created_at, updated_at,
			deleted_at, deleted_by
		FROM photos
		WHERE upload_token = $1`

	row := r.db.Pool().QueryRow(ctx, query, token.String())

	return r.scanPhoto(row)
}

func (r *PhotoRepository) Update(ctx context.Context, photo *entity.Photo) error {
	query := `
		UPDATE photos SET
			route_id = $2,
			lane_code = $3,
			latitude = $4,
			longitude = $5,
			sta_value = $6,
			sta_source = $7,
			description = $8,
			tags = $9,
			updated_at = $10
		WHERE photo_id = $1 AND deleted_at IS NULL`

	var staSourceStr *string
	if photo.STASource() != nil {
		str := photo.STASource().String()
		staSourceStr = &str
	}

	result, err := r.db.Pool().Exec(ctx, query,
		photo.ID().String(),
		photo.RouteID(),
		photo.LaneCode(),
		photo.Latitude(),
		photo.Longitude(),
		photo.STAValue(),
		staSourceStr,
		photo.Description(),
		photo.Tags(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update photo: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) SoftDelete(ctx context.Context, id vo.PhotoID, deletedBy string) error {
	query := `
		UPDATE photos SET
			deleted_at = $2,
			deleted_by = $3,
			updated_at = $4
		WHERE photo_id = $1 AND deleted_at IS NULL`

	result, err := r.db.Pool().Exec(ctx, query, id.String(), time.Now(), deletedBy, time.Now())
	if err != nil {
		return fmt.Errorf("failed to soft delete photo: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) HardDelete(ctx context.Context, id vo.PhotoID) error {
	query := `DELETE FROM photos WHERE photo_id = $1`

	result, err := r.db.Pool().Exec(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to hard delete photo: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) Restore(ctx context.Context, id vo.PhotoID) error {
	query := `
		UPDATE photos SET
			deleted_at = NULL,
			deleted_by = NULL,
			updated_at = $2
		WHERE photo_id = $1 AND deleted_at IS NOT NULL`

	result, err := r.db.Pool().Exec(ctx, query, id.String(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to restore photo: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) UpdateSTA(ctx context.Context, id vo.PhotoID, staValue *float64, source *vo.STASource) error {
	query := `
		UPDATE photos SET
			sta_value = $2,
			sta_source = $3,
			updated_at = $4
		WHERE photo_id = $1 AND deleted_at IS NULL`

	var staSourceStr *string
	if source != nil {
		str := source.String()
		staSourceStr = &str
	}

	result, err := r.db.Pool().Exec(ctx, query, id.String(), staValue, staSourceStr, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update STA: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) Browse(ctx context.Context, filter repository.BrowseFilter) (*repository.BrowseResult, error) {
	// Set defaults for pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	offset := (filter.Page - 1) * filter.PerPage

	// Build the WHERE clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIndex := 1

	if filter.RouteID != "" {
		whereClause += fmt.Sprintf(" AND route_id = $%d", argIndex)
		args = append(args, filter.RouteID)
		argIndex++
	}

	if filter.STAStart != nil {
		whereClause += fmt.Sprintf(" AND sta_value >= $%d", argIndex)
		args = append(args, *filter.STAStart)
		argIndex++
	}

	if filter.STAEnd != nil {
		whereClause += fmt.Sprintf(" AND sta_value <= $%d", argIndex)
		args = append(args, *filter.STAEnd)
		argIndex++
	}

	if filter.Lane != nil {
		whereClause += fmt.Sprintf(" AND lane_code = $%d", argIndex)
		args = append(args, *filter.Lane)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM photos %s", whereClause)
	var totalCount int64
	if err := r.db.Pool().QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count photos: %w", err)
	}

	// Query photos with pagination
	query := fmt.Sprintf(`
		SELECT photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, gcs_object_name,
			file_format, file_size_bytes, original_filename,
			description, tags,
			upload_token, upload_status, retry_count, uploaded_by, uploaded_at,
			created_at, updated_at,
			deleted_at, deleted_by
		FROM photos
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to browse photos: %w", err)
	}
	defer rows.Close()

	photos, err := r.scanPhotos(rows)
	if err != nil {
		return nil, err
	}

	return &repository.BrowseResult{
		Photos:     photos,
		TotalCount: totalCount,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
	}, nil
}

func (r *PhotoRepository) Search(ctx context.Context, filter repository.SearchFilter) (*repository.BrowseResult, error) {
	// Set defaults for pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}
	if filter.PerPage > 100 {
		filter.PerPage = 100
	}

	offset := (filter.Page - 1) * filter.PerPage

	// Build the WHERE clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIndex := 1

	// Route IDs filter (IN clause)
	if len(filter.RouteIDs) > 0 {
		placeholders := make([]string, len(filter.RouteIDs))
		for i, routeID := range filter.RouteIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, routeID)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND route_id IN (%s)", strings.Join(placeholders, ","))
	}

	// STA ranges filter
	if len(filter.STARanges) > 0 {
		staConditions := make([]string, len(filter.STARanges))
		for i, sr := range filter.STARanges {
			staConditions[i] = fmt.Sprintf("(sta_value >= $%d AND sta_value <= $%d)", argIndex, argIndex+1)
			args = append(args, sr.Start, sr.End)
			argIndex += 2
		}
		whereClause += fmt.Sprintf(" AND (%s)", strings.Join(staConditions, " OR "))
	}

	// Lanes filter (IN clause)
	if len(filter.Lanes) > 0 {
		placeholders := make([]string, len(filter.Lanes))
		for i, lane := range filter.Lanes {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, lane)
			argIndex++
		}
		whereClause += fmt.Sprintf(" AND lane_code IN (%s)", strings.Join(placeholders, ","))
	}

	// Date range filter
	if filter.DateStart != nil {
		whereClause += fmt.Sprintf(" AND uploaded_at >= $%d", argIndex)
		args = append(args, *filter.DateStart)
		argIndex++
	}

	if filter.DateEnd != nil {
		whereClause += fmt.Sprintf(" AND uploaded_at <= $%d", argIndex)
		args = append(args, *filter.DateEnd)
		argIndex++
	}

	// Tags filter (array overlap)
	if len(filter.Tags) > 0 {
		whereClause += fmt.Sprintf(" AND tags @> $%d", argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM photos %s", whereClause)
	var totalCount int64
	if err := r.db.Pool().QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count photos: %w", err)
	}

	// Query photos with pagination
	query := fmt.Sprintf(`
		SELECT photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, gcs_object_name,
			file_format, file_size_bytes, original_filename,
			description, tags,
			upload_token, upload_status, retry_count, uploaded_by, uploaded_at,
			created_at, updated_at,
			deleted_at, deleted_by
		FROM photos
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search photos: %w", err)
	}
	defer rows.Close()

	photos, err := r.scanPhotos(rows)
	if err != nil {
		return nil, err
	}

	return &repository.BrowseResult{
		Photos:     photos,
		TotalCount: totalCount,
		Page:       filter.Page,
		PerPage:    filter.PerPage,
	}, nil
}

func (r *PhotoRepository) Exists(ctx context.Context, id vo.PhotoID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM photos WHERE photo_id = $1 AND deleted_at IS NULL)`

	var exists bool
	if err := r.db.Pool().QueryRow(ctx, query, id.String()).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check photo existence: %w", err)
	}

	return exists, nil
}

func (r *PhotoRepository) UpdateUploadStatus(ctx context.Context, id vo.PhotoID, status vo.UploadStatus) error {
	query := `
		UPDATE photos SET
			upload_status = $2,
			updated_at = $3
		WHERE photo_id = $1 AND deleted_at IS NULL`

	result, err := r.db.Pool().Exec(ctx, query, id.String(), status.String(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to update upload status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) IncrementRetryCount(ctx context.Context, id vo.PhotoID) error {
	// First check if the photo exists and get current retry count
	query := `
		SELECT retry_count FROM photos
		WHERE photo_id = $1 AND deleted_at IS NULL`

	var currentCount int
	err := r.db.Pool().QueryRow(ctx, query, id.String()).Scan(&currentCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return repository.ErrPhotoNotFound
		}
		return fmt.Errorf("failed to get retry count: %w", err)
	}

	// Check if max retries exceeded
	if currentCount >= 5 {
		return repository.ErrRetryLimitExceeded
	}

	// Atomically increment the retry count
	updateQuery := `
		UPDATE photos SET
			retry_count = retry_count + 1,
			updated_at = $2
		WHERE photo_id = $1 AND deleted_at IS NULL AND retry_count < 5`

	result, err := r.db.Pool().Exec(ctx, updateQuery, id.String(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	if result.RowsAffected() == 0 {
		// This can happen if another request incremented between our check and update
		return repository.ErrRetryLimitExceeded
	}

	return nil
}

func (r *PhotoRepository) FindPendingByIDAndAPIKey(ctx context.Context, id vo.PhotoID, apiKeyID string) (*entity.Photo, error) {
	query := `
		SELECT photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, gcs_object_name,
			file_format, file_size_bytes, original_filename,
			description, tags,
			upload_token, upload_status, retry_count, uploaded_by, uploaded_at,
			created_at, updated_at,
			deleted_at, deleted_by
		FROM photos
		WHERE photo_id = $1 AND deleted_at IS NULL AND upload_status = 'pending'`

	row := r.db.Pool().QueryRow(ctx, query, id.String())

	var p photoRow
	err := row.Scan(
		&p.PhotoID,
		&p.RouteID,
		&p.LaneCode,
		&p.Latitude,
		&p.Longitude,
		&p.StaValue,
		&p.StaSource,
		&p.GCSObjectName,
		&p.FileFormat,
		&p.FileSizeBytes,
		&p.OriginalFilename,
		&p.Description,
		&p.Tags,
		&p.UploadToken,
		&p.UploadStatus,
		&p.RetryCount,
		&p.UploadedBy,
		&p.UploadedAt,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
		&p.DeletedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrPhotoNotFound
		}
		return nil, fmt.Errorf("failed to scan photo: %w", err)
	}

	photo, err := p.toEntity()
	if err != nil {
		return nil, err
	}

	// Verify API key ownership
	if photo.UploadedBy() != apiKeyID {
		return nil, repository.ErrPhotoNotOwned
	}

	return photo, nil
}

func (r *PhotoRepository) scanPhoto(row pgx.Row) (*entity.Photo, error) {
	var p photoRow
	err := row.Scan(
		&p.PhotoID,
		&p.RouteID,
		&p.LaneCode,
		&p.Latitude,
		&p.Longitude,
		&p.StaValue,
		&p.StaSource,
		&p.GCSObjectName,
		&p.FileFormat,
		&p.FileSizeBytes,
		&p.OriginalFilename,
		&p.Description,
		&p.Tags,
		&p.UploadToken,
		&p.UploadStatus,
		&p.RetryCount,
		&p.UploadedBy,
		&p.UploadedAt,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
		&p.DeletedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrPhotoNotFound
		}
		return nil, fmt.Errorf("failed to scan photo: %w", err)
	}

	return p.toEntity()
}

func (r *PhotoRepository) scanPhotos(rows pgx.Rows) ([]*entity.Photo, error) {
	var photos []*entity.Photo

	for rows.Next() {
		var p photoRow
		err := rows.Scan(
			&p.PhotoID,
			&p.RouteID,
			&p.LaneCode,
			&p.Latitude,
			&p.Longitude,
			&p.StaValue,
			&p.StaSource,
			&p.GCSObjectName,
			&p.FileFormat,
			&p.FileSizeBytes,
			&p.OriginalFilename,
			&p.Description,
			&p.Tags,
			&p.UploadToken,
			&p.UploadStatus,
			&p.RetryCount,
			&p.UploadedBy,
			&p.UploadedAt,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.DeletedAt,
			&p.DeletedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan photo row: %w", err)
		}

		photo, err := p.toEntity()
		if err != nil {
			return nil, err
		}

		photos = append(photos, photo)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating photo rows: %w", err)
	}

	return photos, nil
}
