package reviewusecase

import (
	"context"
	"time"

	"zentora-service/internal/domain/review"
	reviewrepo "zentora-service/internal/repository/review"
)

type Service struct {
	repo reviewrepo.Repository
}

func NewService(repo reviewrepo.Repository) *Service {
	return &Service{repo: repo}
}

// Business rules:
// - logged-in users only (handler enforces)
// - only one review per order item (we bind to order_item_id)
// - within 7 days after order completion
// - no editing
func (s *Service) AddReview(ctx context.Context, userID int64, req *review.CreateRequest) (*review.Review, error) {
	if userID <= 0 || req == nil || req.ProductID <= 0 || req.OrderItemID <= 0 {
		return nil, review.ErrInvalidInput
	}
	if req.Rating < 1 || req.Rating > 5 {
		return nil, review.ErrInvalidInput
	}

	info, err := s.repo.GetOrderItemForReview(ctx, req.OrderItemID)
	if err != nil {
		return nil, err
	}

	// ownership
	if info.OrderUserID == nil || *info.OrderUserID != userID {
		return nil, review.ErrForbidden
	}

	// ensure order item matches product
	if info.ProductID != req.ProductID {
		return nil, review.ErrInvalidInput
	}

	// completion check (you must confirm your "completed" statuses; using delivered as example)
	if info.OrderStatus != "delivered" && info.OrderStatus != "completed" {
		return nil, review.ErrOrderNotCompleted
	}

	completedAt, err := time.Parse(time.RFC3339, info.CompletedAt)
	if err != nil {
		return nil, review.ErrInvalidInput
	}

	if time.Since(completedAt) > 7*24*time.Hour {
		return nil, review.ErrReviewWindowEnded
	}

	oid := req.OrderItemID
	rv := &review.Review{
		UserID:            userID,
		ProductID:         req.ProductID,
		OrderItemID:       &oid,
		Rating:            req.Rating,
		Comment:           req.Comment,
		IsVerifiedPurchase: true,
	}

	if err := s.repo.Create(ctx, rv); err != nil {
		return nil, err
	}

	return rv, nil
}

func (s *Service) ListProductReviews(ctx context.Context, productID int64, limit, offset int) ([]review.Review, int64, error) {
	if productID <= 0 {
		return nil, 0, review.ErrInvalidInput
	}
	return s.repo.ListByProduct(ctx, productID, limit, offset)
}