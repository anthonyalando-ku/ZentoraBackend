package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"zentora-service/internal/domain/cart"
	cartrepo "zentora-service/internal/repository/cart"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CartRepository struct {
	db *pgxpool.Pool
}

func NewCartRepository(db *pgxpool.Pool) *CartRepository {
	return &CartRepository{db: db}
}

var _ cartrepo.Repository = (*CartRepository)(nil)

func scanCart(row pgx.Row) (*cart.Cart, error) {
	var c cart.Cart
	if err := row.Scan(&c.ID, &c.UserID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func scanCartItem(row pgx.Row) (*cart.CartItem, error) {
	var it cart.CartItem
	if err := row.Scan(
		&it.ID,
		&it.CartID,
		&it.ProductID,
		&it.VariantID,
		&it.Quantity,
		&it.PriceAtAdded,
		&it.AddedAt,
	); err != nil {
		return nil, err
	}
	return &it, nil
}

// Production recommendation (schema): enforce single active cart at DB-level.
//
//   CREATE UNIQUE INDEX ux_carts_user_active ON carts(user_id) WHERE status='active';
//
// With that index, even if two concurrent creates happen, one fails with unique violation.
// This code also uses SERIALIZABLE + FOR UPDATE to prevent it.

func (r *CartRepository) GetOrCreateActiveCartForUser(ctx context.Context, userID int64) (*cart.Cart, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid userID")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const findQ = `
		SELECT id, user_id, status, created_at, updated_at
		FROM carts
		WHERE user_id = $1 AND status = 'active'
		FOR UPDATE
	`
	c, err := scanCart(tx.QueryRow(ctx, findQ, userID))
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
		return c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("find active cart: %w", err)
	}

	const insQ = `
		INSERT INTO carts (user_id, status)
		VALUES ($1, 'active')
		RETURNING id, user_id, status, created_at, updated_at
	`
	c, err = scanCart(tx.QueryRow(ctx, insQ, userID))
	if err != nil {
		return nil, fmt.Errorf("insert cart: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return c, nil
}

func (r *CartRepository) GetActiveCartWithItemsForUser(ctx context.Context, userID int64) (*cart.Cart, error) {
	const cartQ = `
		SELECT id, user_id, status, created_at, updated_at
		FROM carts
		WHERE user_id = $1 AND status = 'active'
		ORDER BY id DESC
		LIMIT 1
	`
	c, err := scanCart(r.db.QueryRow(ctx, cartQ, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active cart: %w", err)
	}

	items, err := r.GetCartItems(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	c.Items = items
	return c, nil
}

func (r *CartRepository) GetCartItems(ctx context.Context, cartID int64) ([]cart.CartItem, error) {
	const q = `
		SELECT id, cart_id, product_id, variant_id, quantity, price_at_added, added_at
		FROM cart_items
		WHERE cart_id = $1
		ORDER BY id ASC
	`
	rows, err := r.db.Query(ctx, q, cartID)
	if err != nil {
		return nil, fmt.Errorf("list cart items: %w", err)
	}
	defer rows.Close()

	out := make([]cart.CartItem, 0, 16)
	for rows.Next() {
		it, err := scanCartItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		out = append(out, *it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list cart items rows: %w", err)
	}

	return out, nil
}

// Since variant is REQUIRED, we can safely rely on UNIQUE(cart_id, variant_id).
func (r *CartRepository) UpsertCartItem(ctx context.Context, cartID int64, in cart.UpsertCartItemInput) (*cart.CartItem, error) {
	if cartID <= 0 {
		return nil, fmt.Errorf("invalid cartID")
	}
	if in.ProductID <= 0 {
		return nil, fmt.Errorf("invalid productID")
	}
	if in.VariantID <= 0 {
		return nil, fmt.Errorf("invalid variantID")
	}
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be > 0")
	}

	const q = `
		INSERT INTO cart_items (cart_id, product_id, variant_id, quantity, price_at_added, added_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (cart_id, variant_id)
		DO UPDATE SET
			product_id = EXCLUDED.product_id,
			quantity = EXCLUDED.quantity,
			price_at_added = EXCLUDED.price_at_added,
			added_at = NOW()
		RETURNING id, cart_id, product_id, variant_id, quantity, price_at_added, added_at
	`
	it, err := scanCartItem(r.db.QueryRow(ctx, q, cartID, in.ProductID, in.VariantID, in.Quantity, in.PriceAtAdded))
	if err != nil {
		return nil, fmt.Errorf("upsert cart item: %w", err)
	}

	_, _ = r.db.Exec(ctx, `UPDATE carts SET updated_at = $2 WHERE id = $1`, cartID, time.Now().UTC())
	return it, nil
}

func (r *CartRepository) RemoveCartItem(ctx context.Context, cartID int64, itemID int64) error {
	if cartID <= 0 || itemID <= 0 {
		return fmt.Errorf("invalid ids")
	}

	ct, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE id = $1 AND cart_id = $2`, itemID, cartID)
	if err != nil {
		return fmt.Errorf("delete cart item: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	_, _ = r.db.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cartID)
	return nil
}

func (r *CartRepository) ClearCart(ctx context.Context, cartID int64) error {
	if cartID <= 0 {
		return fmt.Errorf("invalid cartID")
	}
	_, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}
	_, _ = r.db.Exec(ctx, `UPDATE carts SET updated_at = NOW() WHERE id = $1`, cartID)
	return nil
}

func (r *CartRepository) MarkCartConverted(ctx context.Context, cartID int64) error {
	ct, err := r.db.Exec(ctx, `UPDATE carts SET status='converted', updated_at=NOW() WHERE id=$1 AND status='active'`, cartID)
	if err != nil {
		return fmt.Errorf("mark cart converted: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}