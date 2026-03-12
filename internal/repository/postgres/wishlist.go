package postgres

import (
	"context"
	"errors"
	"fmt"

	"zentora-service/internal/domain/wishlist"
	wishlistrepo "zentora-service/internal/repository/wishlist"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WishlistRepository struct {
	db *pgxpool.Pool
}

func NewWishlistRepository(db *pgxpool.Pool) *WishlistRepository {
	return &WishlistRepository{db: db}
}

var _ wishlistrepo.Repository = (*WishlistRepository)(nil)

func (r *WishlistRepository) GetOrCreateByUser(ctx context.Context, userID int64) (*wishlist.Wishlist, error) {
	if userID <= 0 {
		return nil, wishlist.ErrInvalidInput
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock user wishlist row to avoid duplicate create under concurrency
	const sel = `
		SELECT id, user_id, created_at
		FROM wishlists
		WHERE user_id = $1
		FOR UPDATE
	`
	var w wishlist.Wishlist
	err = tx.QueryRow(ctx, sel, userID).Scan(&w.ID, &w.UserID, &w.CreatedAt)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
		return &w, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("select wishlist: %w", err)
	}

	const ins = `
		INSERT INTO wishlists (user_id)
		VALUES ($1)
		RETURNING id, user_id, created_at
	`
	err = tx.QueryRow(ctx, ins, userID).Scan(&w.ID, &w.UserID, &w.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert wishlist: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &w, nil
}

func (r *WishlistRepository) GetByUserWithItems(ctx context.Context, userID int64) (*wishlist.Wishlist, error) {
	if userID <= 0 {
		return nil, wishlist.ErrInvalidInput
	}

	const selW = `
		SELECT id, user_id, created_at
		FROM wishlists
		WHERE user_id = $1
	`
	var w wishlist.Wishlist
	if err := r.db.QueryRow(ctx, selW, userID).Scan(&w.ID, &w.UserID, &w.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get wishlist: %w", err)
	}

	const selItems = `
		SELECT wishlist_id, product_id, variant_id, added_at
		FROM wishlist_items
		WHERE wishlist_id = $1
		ORDER BY added_at DESC
	`
	rows, err := r.db.Query(ctx, selItems, w.ID)
	if err != nil {
		return nil, fmt.Errorf("list wishlist items: %w", err)
	}
	defer rows.Close()

	items := make([]wishlist.WishlistItem, 0, 16)
	for rows.Next() {
		var it wishlist.WishlistItem
		if err := rows.Scan(&it.WishlistID, &it.ProductID, &it.VariantID, &it.AddedAt); err != nil {
			return nil, fmt.Errorf("scan wishlist item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("wishlist items rows: %w", err)
	}

	w.Items = items
	return &w, nil
}

func (r *WishlistRepository) AddItem(ctx context.Context, userID int64, productID int64, variantID int64) error {
	if userID <= 0 || productID <= 0 || variantID <= 0 {
		return wishlist.ErrInvalidInput
	}

	w, err := r.GetOrCreateByUser(ctx, userID)
	if err != nil {
		return err
	}

	const q = `
		INSERT INTO wishlist_items (wishlist_id, product_id, variant_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (wishlist_id, product_id, variant_id) DO NOTHING
	`
	_, err = r.db.Exec(ctx, q, w.ID, productID, variantID)
	if err != nil {
		return fmt.Errorf("add wishlist item: %w", err)
	}
	return nil
}

func (r *WishlistRepository) RemoveItem(ctx context.Context, userID int64, productID int64, variantID int64) error {
	if userID <= 0 || productID <= 0 || variantID <= 0 {
		return wishlist.ErrInvalidInput
	}

	const q = `
		DELETE FROM wishlist_items wi
		USING wishlists w
		WHERE w.user_id = $1
		  AND wi.wishlist_id = w.id
		  AND wi.product_id = $2
		  AND wi.variant_id = $3
	`
	ct, err := r.db.Exec(ctx, q, userID, productID, variantID)
	if err != nil {
		return fmt.Errorf("remove wishlist item: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return wishlist.ErrNotFound
	}
	return nil
}

func (r *WishlistRepository) Clear(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return wishlist.ErrInvalidInput
	}

	const q = `
		DELETE FROM wishlist_items wi
		USING wishlists w
		WHERE w.user_id = $1
		  AND wi.wishlist_id = w.id
	`
	_, err := r.db.Exec(ctx, q, userID)
	if err != nil {
		return fmt.Errorf("clear wishlist: %w", err)
	}
	return nil
}