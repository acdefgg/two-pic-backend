package repository

import (
	"context"
	"fmt"

	"sync-photo-backend/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, code, token, push_token, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query, user.ID, user.Code, user.Token, user.PushToken, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, code, token, push_token, created_at
		FROM users
		WHERE id = $1
	`
	var user models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Code, &user.Token, &user.PushToken, &user.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetByCode retrieves a user by code
func (r *UserRepository) GetByCode(ctx context.Context, code string) (*models.User, error) {
	query := `
		SELECT id, code, token, push_token, created_at
		FROM users
		WHERE code = $1
	`
	var user models.User
	err := r.db.QueryRow(ctx, query, code).Scan(
		&user.ID, &user.Code, &user.Token, &user.PushToken, &user.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user by code: %w", err)
	}
	return &user, nil
}

// CodeExists checks if a code already exists
func (r *UserRepository) CodeExists(ctx context.Context, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE code = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check code existence: %w", err)
	}
	return exists, nil
}

// UpdatePushToken updates the push token for a user
func (r *UserRepository) UpdatePushToken(ctx context.Context, userID string, pushToken *string) error {
	query := `UPDATE users SET push_token = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, pushToken, userID)
	if err != nil {
		return fmt.Errorf("failed to update push token: %w", err)
	}
	return nil
}
