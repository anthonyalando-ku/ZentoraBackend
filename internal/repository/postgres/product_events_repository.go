package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	xerrors "zentora-service/internal/pkg/errors"
)

type ProductEventType string

const (
	ProductEventView      ProductEventType = "view"
	ProductEventAddToCart ProductEventType = "add_to_cart"
	ProductEventPurchase  ProductEventType = "purchase"
	ProductEventWishlist  ProductEventType = "wishlist"
)

type CreateProductEventParams struct {
	ProductID  int64
	UserID     *int64
	SessionID  *string
	EventType  ProductEventType
	Quantity   int
}

type ProductEventsRepository struct {
	db *pgxpool.Pool
}

func NewProductEventsRepository(db *pgxpool.Pool) *ProductEventsRepository {
	return &ProductEventsRepository{db: db}
}

// Create inserts a product event. It always runs inside a transaction.
func (r *ProductEventsRepository) Create(ctx context.Context, p CreateProductEventParams) error {
	if p.ProductID <= 0 {
		return fmt.Errorf("product_id must be > 0")
	}
	if p.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if p.Quantity <= 0 {
		p.Quantity = 1
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return xerrors.Wrap(err, "begin tx")
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const q = `
		INSERT INTO product_events (product_id, user_id, session_id, event_type, quantity)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, execErr := tx.Exec(ctx, q, p.ProductID, p.UserID, p.SessionID, string(p.EventType), p.Quantity)
	if execErr != nil {
		if pgErr, ok := execErr.(*pgconn.PgError); ok {
			return xerrors.Wrap(pgErr, "insert product_event")
		}
		return xerrors.Wrap(execErr, "insert product_event")
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return xerrors.Wrap(commitErr, "commit tx")
	}

	return nil
}