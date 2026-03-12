package reviewrepo

import (
	"context"

	"zentora-service/internal/domain/review"
)

type Repository interface {
	Create(ctx context.Context, r *review.Review) error
	ListByProduct(ctx context.Context, productID int64, limit, offset int) ([]review.Review, int64, error)

	// for validation rules (ownership, timing, product match)
	GetOrderItemForReview(ctx context.Context, orderItemID int64) (*OrderItemReviewInfo, error)
}

type OrderItemReviewInfo struct {
	OrderItemID int64
	OrderID     int64
	OrderUserID *int64
	ProductID   int64

	OrderStatus string
	OrderUpdatedAt  *string // optional; kept simple if you don’t store completion timestamp yet
	OrderCreatedAt  string  // ISO-ish; repo can map to time in impl if you want

	// Best: store completed_at in orders, but schema doesn't have it.
	// We'll use orders.updated_at as completion timestamp as a practical proxy.
	CompletedAt string
}