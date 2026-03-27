package postgres

import (
	"context"
	"encoding/json"
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
	PhotoID               string
	RouteID               string
	LaneCode              string
	Latitude              float64
	Longitude             float64
	StaValue              float64
	StaSource             string
	OriginalPath          string
	ThumbnailSmallPath    *string
	ThumbnailMediumPath   *string
	ThumbnailLargePath    *string
	FileFormat            string
	FileSizeBytes         int64
	OriginalFilename      *string
	ExifData              []byte // JSON
	Description           *string
	Tags                  []string
	UploadToken           string
	UploadStatus          string
	UploadedBy            string
	UploadedAt            time.Time
	Status                string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ProcessingCompletedAt *time.Time
	DeletedAt             *time.Time
	DeletedBy             *string
}

func (r *photoRow) toEntity() (*entity.Photo, error) {
	var exifData *entity.EXIFData
	if r.ExifData != nil {
		exif := &entity.EXIFData{}
		if err := json.Unmarshal(r.ExifData, exif); err != nil {
			return nil, fmt.Errorf("failed to unmarshal EXIF data: %w", err)
		}
		exifData = exif
	}

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

	photoStatus, err := vo.ParsePhotoStatus(r.Status)
	if err != nil {
		return nil, fmt.Errorf("invalid photo status: %w", err)
	}

	staSource, err := vo.ParseSTASource(r.StaSource)
	if err != nil {
		return nil, fmt.Errorf("invalid STA source: %w", err)
	}

	params := entity.PhotoRowParams{
		ID:                    photoID,
		RouteID:               r.RouteID,
		LaneCode:              r.LaneCode,
		Latitude:              r.Latitude,
		Longitude:             r.Longitude,
		StaValue:              r.StaValue,
		StaSource:             staSource,
		OriginalPath:          r.OriginalPath,
		ThumbnailSmallPath:    r.ThumbnailSmallPath,
		ThumbnailMediumPath:   r.ThumbnailMediumPath,
		ThumbnailLargePath:    r.ThumbnailLargePath,
		FileFormat:            fileFormat,
		FileSizeBytes:         r.FileSizeBytes,
		OriginalFilename:      r.OriginalFilename,
		EXIFData:              exifData,
		Description:           r.Description,
		Tags:                  r.Tags,
		UploadToken:           uploadToken,
		UploadStatus:          uploadStatus,
		UploadedBy:            r.UploadedBy,
		UploadedAt:            r.UploadedAt,
		Status:                photoStatus,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
		ProcessingCompletedAt: r.ProcessingCompletedAt,
		DeletedAt:             r.DeletedAt,
		DeletedBy:             r.DeletedBy,
	}

	return entity.NewPhotoFromRepository(params), nil
}

func (r *PhotoRepository) Create(ctx context.Context, photo *entity.Photo) error {
	exifDataJSON, err := json.Marshal(photo.EXIFData())
	if err != nil {
		return fmt.Errorf("failed to marshal EXIF data: %w", err)
	}

	query := `
		INSERT INTO photos (
			photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, original_path,
			thumbnail_small_path, thumbnail_medium_path, thumbnail_large_path,
			file_format, file_size_bytes, original_filename,
			exif_data, description, tags,
			upload_token, upload_status, uploaded_by, uploaded_at,
			status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19, $20, $21,
			$22, $23, $24
		)`

	_, err = r.db.Pool().Exec(ctx, query,
		photo.ID().String(),
		photo.RouteID(),
		photo.LaneCode(),
		photo.Latitude(),
		photo.Longitude(),
		photo.STAValue(),
		photo.STASource().String(),
		photo.OriginalPath(),
		photo.ThumbnailSmallPath(),
		photo.ThumbnailMediumPath(),
		photo.ThumbnailLargePath(),
		photo.FileFormat().String(),
		photo.FileSizeBytes(),
		photo.OriginalFilename(),
		exifDataJSON,
		photo.Description(),
		photo.Tags(),
		photo.UploadToken().String(),
		photo.UploadStatus().String(),
		photo.UploadedBy(),
		photo.UploadedAt(),
		photo.Status().String(),
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
			sta_value, sta_source, original_path,
			thumbnail_small_path, thumbnail_medium_path, thumbnail_large_path,
			file_format, file_size_bytes, original_filename,
			exif_data, description, tags,
			upload_token, upload_status, uploaded_by, uploaded_at,
			status, created_at, updated_at,
			processing_completed_at, deleted_at, deleted_by
		FROM photos
		WHERE photo_id = $1 AND deleted_at IS NULL`

	row := r.db.Pool().QueryRow(ctx, query, id.String())

	return r.scanPhoto(row)
}

func (r *PhotoRepository) GetByUploadToken(ctx context.Context, token vo.UploadToken) (*entity.Photo, error) {
	query := `
		SELECT photo_id, route_id, lane_code, latitude, longitude,
			sta_value, sta_source, original_path,
			thumbnail_small_path, thumbnail_medium_path, thumbnail_large_path,
			file_format, file_size_bytes, original_filename,
			exif_data, description, tags,
			upload_token, upload_status, uploaded_by, uploaded_at,
			status, created_at, updated_at,
			processing_completed_at, deleted_at, deleted_by
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
			thumbnail_small_path = $8,
			thumbnail_medium_path = $9,
			thumbnail_large_path = $10,
			description = $11,
			tags = $12,
			status = $13,
			updated_at = $14,
			processing_completed_at = $15
		WHERE photo_id = $1 AND deleted_at IS NULL`

	result, err := r.db.Pool().Exec(ctx, query,
		photo.ID().String(),
		photo.RouteID(),
		photo.LaneCode(),
		photo.Latitude(),
		photo.Longitude(),
		photo.STAValue(),
		photo.STASource().String(),
		photo.ThumbnailSmallPath(),
		photo.ThumbnailMediumPath(),
		photo.ThumbnailLargePath(),
		photo.Description(),
		photo.Tags(),
		photo.Status().String(),
		time.Now(),
		photo.ProcessingCompletedAt(),
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

func (r *PhotoRepository) UpdateProcessingStatus(ctx context.Context, id vo.PhotoID, status vo.PhotoStatus, thumbnailPaths entity.ThumbnailPaths) error {
	smallPath := thumbnailPaths.Small
	mediumPath := thumbnailPaths.Medium
	largePath := thumbnailPaths.Large

	query := `
		UPDATE photos SET
			status = $2,
			thumbnail_small_path = $3,
			thumbnail_medium_path = $4,
			thumbnail_large_path = $5,
			processing_completed_at = $6,
			updated_at = $7
		WHERE photo_id = $1 AND deleted_at IS NULL`

	result, err := r.db.Pool().Exec(ctx, query,
		id.String(),
		status.String(),
		smallPath,
		mediumPath,
		largePath,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update processing status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) UpdateEXIFData(ctx context.Context, id vo.PhotoID, exifData *entity.EXIFData) error {
	exifDataJSON, err := json.Marshal(exifData)
	if err != nil {
		return fmt.Errorf("failed to marshal EXIF data: %w", err)
	}

	query := `
		UPDATE photos SET
			exif_data = $2,
			updated_at = $3
		WHERE photo_id = $1 AND deleted_at IS NULL`

	result, err := r.db.Pool().Exec(ctx, query, id.String(), exifDataJSON, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update EXIF data: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrPhotoNotFound
	}

	return nil
}

func (r *PhotoRepository) UpdateSTA(ctx context.Context, id vo.PhotoID, staValue float64, source vo.STASource) error {
	query := `
		UPDATE photos SET
			sta_value = $2,
			sta_source = $3,
			updated_at = $4
		WHERE photo_id = $1 AND deleted_at IS NULL`

	result, err := r.db.Pool().Exec(ctx, query, id.String(), staValue, source.String(), time.Now())
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
			sta_value, sta_source, original_path,
			thumbnail_small_path, thumbnail_medium_path, thumbnail_large_path,
			file_format, file_size_bytes, original_filename,
			exif_data, description, tags,
			upload_token, upload_status, uploaded_by, uploaded_at,
			status, created_at, updated_at,
			processing_completed_at, deleted_at, deleted_by
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

	// EXIF GPS filter
	if filter.HasEXIFGPS != nil {
		if *filter.HasEXIFGPS {
			whereClause += " AND exif_data IS NOT NULL AND exif_data->>'gps_latitude' IS NOT NULL"
		} else {
			whereClause += " AND (exif_data IS NULL OR exif_data->>'gps_latitude' IS NULL)"
		}
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
			sta_value, sta_source, original_path,
			thumbnail_small_path, thumbnail_medium_path, thumbnail_large_path,
			file_format, file_size_bytes, original_filename,
			exif_data, description, tags,
			upload_token, upload_status, uploaded_by, uploaded_at,
			status, created_at, updated_at,
			processing_completed_at, deleted_at, deleted_by
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
		&p.OriginalPath,
		&p.ThumbnailSmallPath,
		&p.ThumbnailMediumPath,
		&p.ThumbnailLargePath,
		&p.FileFormat,
		&p.FileSizeBytes,
		&p.OriginalFilename,
		&p.ExifData,
		&p.Description,
		&p.Tags,
		&p.UploadToken,
		&p.UploadStatus,
		&p.UploadedBy,
		&p.UploadedAt,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.ProcessingCompletedAt,
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
			&p.OriginalPath,
			&p.ThumbnailSmallPath,
			&p.ThumbnailMediumPath,
			&p.ThumbnailLargePath,
			&p.FileFormat,
			&p.FileSizeBytes,
			&p.OriginalFilename,
			&p.ExifData,
			&p.Description,
			&p.Tags,
			&p.UploadToken,
			&p.UploadStatus,
			&p.UploadedBy,
			&p.UploadedAt,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.ProcessingCompletedAt,
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
