package repository

import (
	"context"
	"fmt"

	"sync-photo-backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PhotoRepository handles database operations for photos
type PhotoRepository struct {
	db *pgxpool.Pool
}

// NewPhotoRepository creates a new photo repository
func NewPhotoRepository(db *pgxpool.Pool) *PhotoRepository {
	return &PhotoRepository{db: db}
}

// Create creates a new photo
func (r *PhotoRepository) Create(ctx context.Context, photo *models.Photo) error {
	query := `
		INSERT INTO photos (id, pair_id, user_id, s3_url, taken_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		photo.ID, photo.PairID, photo.UserID, photo.S3URL, photo.TakenAt, photo.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create photo: %w", err)
	}
	return nil
}

// GetByID retrieves a photo by ID
func (r *PhotoRepository) GetByID(ctx context.Context, id string) (*models.Photo, error) {
	query := `
		SELECT id, pair_id, user_id, s3_url, taken_at, created_at
		FROM photos
		WHERE id = $1
	`
	var photo models.Photo
	err := r.db.QueryRow(ctx, query, id).Scan(
		&photo.ID, &photo.PairID, &photo.UserID, &photo.S3URL,
		&photo.TakenAt, &photo.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("photo not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get photo: %w", err)
	}
	return &photo, nil
}

// GetByPairID retrieves photos by pair ID with pagination
func (r *PhotoRepository) GetByPairID(ctx context.Context, pairID string, limit, offset int) ([]*models.Photo, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM photos WHERE pair_id = $1`
	var total int
	err := r.db.QueryRow(ctx, countQuery, pairID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count photos: %w", err)
	}

	// Get photos
	query := `
		SELECT id, pair_id, user_id, s3_url, taken_at, created_at
		FROM photos
		WHERE pair_id = $1
		ORDER BY taken_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, pairID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get photos: %w", err)
	}
	defer rows.Close()

	var photos []*models.Photo
	for rows.Next() {
		var photo models.Photo
		err := rows.Scan(
			&photo.ID, &photo.PairID, &photo.UserID, &photo.S3URL,
			&photo.TakenAt, &photo.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan photo: %w", err)
		}
		photos = append(photos, &photo)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating photos: %w", err)
	}

	return photos, total, nil
}

// UpdateS3URL updates the S3 URL for a photo
func (r *PhotoRepository) UpdateS3URL(ctx context.Context, photoID, s3URL string) error {
	query := `UPDATE photos SET s3_url = $1 WHERE id = $2`
	result, err := r.db.Exec(ctx, query, s3URL, photoID)
	if err != nil {
		return fmt.Errorf("failed to update photo s3_url: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("photo not found")
	}
	return nil
}
