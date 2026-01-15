package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"sync-photo-backend/internal/models"
	"sync-photo-backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	codeLength = 6
	codeChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	jwtExpDays = 365
)

// UserService handles user-related business logic
type UserService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

// GenerateUniqueCode generates a unique 6-character code
func (s *UserService) GenerateUniqueCode(ctx context.Context) (string, error) {
	maxAttempts := 10
	for i := 0; i < maxAttempts; i++ {
		code := generateCode()
		exists, err := s.userRepo.CodeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("failed to check code existence: %w", err)
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxAttempts)
}

// generateCode generates a random 6-character code
func generateCode() string {
	code := make([]byte, codeLength)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		code[i] = codeChars[n.Int64()]
	}
	return string(code)
}

// GenerateJWT generates a JWT token for a user
func (s *UserService) GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().AddDate(0, 0, jwtExpDays).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns the user ID
func (s *UserService) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("user_id not found in token")
	}

	return userID, nil
}

// CreateUser creates a new anonymous user
func (s *UserService) CreateUser(ctx context.Context) (*models.User, error) {
	// Generate unique code
	code, err := s.GenerateUniqueCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// Generate user ID
	userID := uuid.New().String()

	// Generate JWT token
	token, err := s.GenerateJWT(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create user
	user := &models.User{
		ID:        userID,
		Code:      code,
		Token:     token,
		CreatedAt: time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}
