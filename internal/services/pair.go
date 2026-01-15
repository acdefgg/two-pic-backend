package services

import (
	"context"
	"fmt"
	"time"

	"sync-photo-backend/internal/models"
	"sync-photo-backend/internal/repository"

	"github.com/google/uuid"
)

// PairService handles pair-related business logic
type PairService struct {
	pairRepo *repository.PairRepository
	userRepo *repository.UserRepository
}

// NewPairService creates a new pair service
func NewPairService(pairRepo *repository.PairRepository, userRepo *repository.UserRepository) *PairService {
	return &PairService{
		pairRepo: pairRepo,
		userRepo: userRepo,
	}
}

// CreatePairRequest represents a request to create a pair
type CreatePairRequest struct {
	PartnerCode string `json:"partner_code"`
}

// CreatePair creates a new pair between two users
func (s *PairService) CreatePair(ctx context.Context, userAID, partnerCode string) (*models.Pair, error) {
	// Validate partner code
	if len(partnerCode) != 6 {
		return nil, fmt.Errorf("partner code must be 6 characters")
	}

	// Get partner user by code
	partnerUser, err := s.userRepo.GetByCode(ctx, partnerCode)
	if err != nil {
		return nil, fmt.Errorf("partner not found: %w", err)
	}

	userBID := partnerUser.ID

	// Check if user is trying to pair with themselves
	if userAID == userBID {
		return nil, fmt.Errorf("cannot create pair with yourself")
	}

	// Check if user A is already in a pair
	hasPair, err := s.pairRepo.UserHasPair(ctx, userAID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if user has pair: %w", err)
	}
	if hasPair {
		return nil, fmt.Errorf("user is already in a pair")
	}

	// Check if partner is already in a pair
	partnerHasPair, err := s.pairRepo.UserHasPair(ctx, userBID)
	if err != nil {
		return nil, fmt.Errorf("failed to check if partner has pair: %w", err)
	}
	if partnerHasPair {
		return nil, fmt.Errorf("partner is already in a pair")
	}

	// Create pair (user_a_id should be lexicographically smaller to ensure consistency)
	if userAID > userBID {
		userAID, userBID = userBID, userAID
	}

	pair := &models.Pair{
		ID:        uuid.New().String(),
		UserAID:   userAID,
		UserBID:   userBID,
		CreatedAt: time.Now(),
	}

	if err := s.pairRepo.Create(ctx, pair); err != nil {
		return nil, fmt.Errorf("failed to create pair: %w", err)
	}

	return pair, nil
}

// DeletePair deletes a pair if the user is a member
func (s *PairService) DeletePair(ctx context.Context, pairID, userID string) error {
	// Get pair
	pair, err := s.pairRepo.GetByID(ctx, pairID)
	if err != nil {
		return fmt.Errorf("pair not found: %w", err)
	}

	// Check if user is a member of the pair
	if pair.UserAID != userID && pair.UserBID != userID {
		return fmt.Errorf("user is not a member of this pair")
	}

	// Delete pair
	if err := s.pairRepo.Delete(ctx, pairID); err != nil {
		return fmt.Errorf("failed to delete pair: %w", err)
	}

	return nil
}

// GetPairByUserID gets the pair for a user
func (s *PairService) GetPairByUserID(ctx context.Context, userID string) (*models.Pair, error) {
	return s.pairRepo.GetByUserID(ctx, userID)
}
