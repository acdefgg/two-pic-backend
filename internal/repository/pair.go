package repository

import (
	"context"
	"fmt"

	"sync-photo-backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PairRepository handles database operations for pairs
type PairRepository struct {
	db *pgxpool.Pool
}

// NewPairRepository creates a new pair repository
func NewPairRepository(db *pgxpool.Pool) *PairRepository {
	return &PairRepository{db: db}
}

// Create creates a new pair
func (r *PairRepository) Create(ctx context.Context, pair *models.Pair) error {
	query := `
		INSERT INTO pairs (id, user_a_id, user_b_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(ctx, query, pair.ID, pair.UserAID, pair.UserBID, pair.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create pair: %w", err)
	}
	return nil
}

// GetByID retrieves a pair by ID
func (r *PairRepository) GetByID(ctx context.Context, id string) (*models.Pair, error) {
	query := `
		SELECT id, user_a_id, user_b_id, created_at
		FROM pairs
		WHERE id = $1
	`
	var pair models.Pair
	err := r.db.QueryRow(ctx, query, id).Scan(
		&pair.ID, &pair.UserAID, &pair.UserBID, &pair.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pair not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get pair: %w", err)
	}
	return &pair, nil
}

// GetByUserID retrieves a pair by user ID
func (r *PairRepository) GetByUserID(ctx context.Context, userID string) (*models.Pair, error) {
	query := `
		SELECT id, user_a_id, user_b_id, created_at
		FROM pairs
		WHERE user_a_id = $1 OR user_b_id = $1
		LIMIT 1
	`
	var pair models.Pair
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&pair.ID, &pair.UserAID, &pair.UserBID, &pair.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pair not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get pair by user id: %w", err)
	}
	return &pair, nil
}

// Delete deletes a pair by ID
func (r *PairRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM pairs WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pair: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("pair not found")
	}
	return nil
}

// UserHasPair checks if a user is already in a pair
func (r *PairRepository) UserHasPair(ctx context.Context, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM pairs WHERE user_a_id = $1 OR user_b_id = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user has pair: %w", err)
	}
	return exists, nil
}
