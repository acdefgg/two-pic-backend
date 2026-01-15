package services

import (
	"context"
	"fmt"
	"time"

	"sync-photo-backend/internal/models"
	"sync-photo-backend/internal/repository"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// PhotoService handles photo-related business logic
type PhotoService struct {
	photoRepo *repository.PhotoRepository
	pairRepo  *repository.PairRepository
	s3Client  *s3.Client
	s3Bucket  string
}

// NewPhotoService creates a new photo service
func NewPhotoService(
	photoRepo *repository.PhotoRepository,
	pairRepo *repository.PairRepository,
	awsRegion, s3Bucket string,
) (*PhotoService, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	return &PhotoService{
		photoRepo: photoRepo,
		pairRepo:  pairRepo,
		s3Client:  s3Client,
		s3Bucket:  s3Bucket,
	}, nil
}

// UploadRequest represents a request to get a pre-signed URL
type UploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// UploadResponse represents the response with pre-signed URL
type UploadResponse struct {
	UploadURL string `json:"upload_url"`
	PhotoID   string `json:"photo_id"`
	ExpiresIn int    `json:"expires_in"`
}

// GetPreSignedURL generates a pre-signed URL for uploading a photo
func (s *PhotoService) GetPreSignedURL(ctx context.Context, userID, filename, contentType string) (*UploadResponse, error) {
	// Get user's pair
	pair, err := s.pairRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user is not in a pair: %w", err)
	}

	// Generate photo ID
	photoID := uuid.New().String()

	// Generate S3 key: {pair_id}/{photo_id}.jpg
	s3Key := fmt.Sprintf("%s/%s.jpg", pair.ID, photoID)

	// Create pre-signed URL request
	presignClient := s3.NewPresignClient(s.s3Client)
	request, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.s3Bucket),
		Key:         aws.String(s3Key),
		ContentType: aws.String(contentType),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = 5 * time.Minute // 5 minutes
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate pre-signed URL: %w", err)
	}

	// Create photo record in DB with placeholder URL (will be updated after upload)
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.s3Bucket, "us-east-1", s3Key)
	photo := &models.Photo{
		ID:        photoID,
		PairID:    pair.ID,
		UserID:    userID,
		S3URL:     s3URL,
		TakenAt:   time.Now(),
		CreatedAt: time.Now(),
	}

	if err := s.photoRepo.Create(ctx, photo); err != nil {
		return nil, fmt.Errorf("failed to create photo record: %w", err)
	}

	return &UploadResponse{
		UploadURL: request.URL,
		PhotoID:   photoID,
		ExpiresIn: 300, // 5 minutes in seconds
	}, nil
}

// UpdatePhotoS3URL updates the S3 URL after upload
func (s *PhotoService) UpdatePhotoS3URL(ctx context.Context, photoID, s3URL string) error {
	return s.photoRepo.UpdateS3URL(ctx, photoID, s3URL)
}

// GetPhotosByPair retrieves photos for a pair with pagination
func (s *PhotoService) GetPhotosByPair(ctx context.Context, userID string, limit, offset int) ([]*models.Photo, int, error) {
	// Get user's pair
	pair, err := s.pairRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("user is not in a pair: %w", err)
	}

	// Validate limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.photoRepo.GetByPairID(ctx, pair.ID, limit, offset)
}
