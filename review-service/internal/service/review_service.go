package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"review-service/internal/models"
	"review-service/internal/repository"
)

var (
	ErrInvalidRating  = errors.New("rating must be between 1 and 5")
	ErrForbidden      = errors.New("forbidden: not the review owner")
	ErrReviewNotFound = repository.ErrReviewNotFound
)

const listLimit = 200

type ReviewService struct {
	repo *repository.ReviewRepository
}

func NewReviewService(repo *repository.ReviewRepository) *ReviewService {
	return &ReviewService{repo: repo}
}

func (s *ReviewService) Create(ctx context.Context, userID, productID string, rating int, text string) (*models.Review, error) {
	if rating < 1 || rating > 5 {
		return nil, ErrInvalidRating
	}
	now := time.Now()
	rv := &models.Review{
		ID:        uuid.NewString(),
		ProductID: productID,
		UserID:    userID,
		Rating:    rating,
		Text:      text,
		CreatedAt: now,
		UpdatedAt: now,
		Media:     []models.ReviewMedia{},
	}
	if err := s.repo.Create(ctx, rv); err != nil {
		return nil, err
	}
	return rv, nil
}

func (s *ReviewService) ListByProduct(ctx context.Context, productID string) ([]*models.Review, error) {
	return s.repo.ListByProduct(ctx, productID, listLimit)
}

// AddMedia attaches an already-saved-to-disk file to a review. Only the
// review's own author may attach media to it.
func (s *ReviewService) AddMedia(ctx context.Context, userID, reviewID, mediaType, url string) (*models.ReviewMedia, error) {
	rv, err := s.repo.FindByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if rv.UserID != userID {
		return nil, ErrForbidden
	}

	m := &models.ReviewMedia{
		ID:        uuid.NewString(),
		ReviewID:  reviewID,
		MediaType: mediaType,
		URL:       url,
		CreatedAt: time.Now(),
	}
	if err := s.repo.AddMedia(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Delete removes a review and its media rows (ON DELETE CASCADE). The
// handler is responsible for best-effort deleting the media files on disk —
// they live under a directory keyed by reviewID, so it doesn't need
// anything back from here to do that.
func (s *ReviewService) Delete(ctx context.Context, userID, reviewID string) error {
	rv, err := s.repo.FindByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if rv.UserID != userID {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, reviewID)
}

// Get is used by the media-upload handler to check ownership before it
// writes the uploaded file to disk.
func (s *ReviewService) Get(ctx context.Context, reviewID string) (*models.Review, error) {
	return s.repo.FindByID(ctx, reviewID)
}
