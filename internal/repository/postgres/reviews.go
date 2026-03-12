package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"zentora-service/internal/domain/review"
	reviewrepo "zentora-service/internal/repository/review"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewRepository struct {
	db *pgxpool.Pool
}

func NewReviewRepository(db *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{db: db}
}

var _ reviewrepo.Repository = (*ReviewRepository)(nil)

func (r *ReviewRepository) Create(ctx context.Context, rv *review.Review) error {
	const q = `
		INSERT INTO reviews (user_id, product_id, order_item_id, rating, comment, is_verified_purchase)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, q,
		rv.UserID,
		rv.ProductID,
		rv.OrderItemID,
		rv.Rating,
		rv.Comment,
		rv.IsVerifiedPurchase,
	).Scan(&rv.ID, &rv.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			// schema has UNIQUE(user_id, product_id)
			return review.ErrConflict
		}
		return fmt.Errorf("create review: %w", err)
	}
	return nil
}

func (r *ReviewRepository) ListByProduct(ctx context.Context, productID int64, limit, offset int) ([]review.Review, int64, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews WHERE product_id=$1`, productID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	const q = `
		SELECT id, user_id, product_id, order_item_id, rating, comment, is_verified_purchase, created_at
		FROM reviews
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, productID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()

	out := make([]review.Review, 0, limit)
	for rows.Next() {
		var rv review.Review
		if err := rows.Scan(
			&rv.ID,
			&rv.UserID,
			&rv.ProductID,
			&rv.OrderItemID,
			&rv.Rating,
			&rv.Comment,
			&rv.IsVerifiedPurchase,
			&rv.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan review: %w", err)
		}
		out = append(out, rv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reviews rows: %w", err)
	}

	return out, total, nil
}

func (r *ReviewRepository) GetOrderItemForReview(ctx context.Context, orderItemID int64) (*reviewrepo.OrderItemReviewInfo, error) {
	// We need: order_items.product_id, orders.user_id, orders.status, and a "completion time".
	// Schema has orders.updated_at, no completed_at. We'll use updated_at when status is 'delivered' or 'completed'.
	const q = `
		SELECT
			oi.id AS order_item_id,
			oi.order_id,
			o.user_id,
			oi.product_id,
			o.status,
			o.updated_at
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.id = $1
	`

	var (
		info     reviewrepo.OrderItemReviewInfo
		userID   *int64
		status   string
		updated  time.Time
	)

	if err := r.db.QueryRow(ctx, q, orderItemID).Scan(
		&info.OrderItemID,
		&info.OrderID,
		&userID,
		&info.ProductID,
		&status,
		&updated,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, review.ErrNotFound
		}
		return nil, fmt.Errorf("get order item review info: %w", err)
	}

	info.OrderUserID = userID
	info.OrderStatus = status
	info.CompletedAt = updated.Format(time.RFC3339)

	return &info, nil
}