package wishlistrepo

import (
	"context"

	"zentora-service/internal/domain/wishlist"
)

type Repository interface {
	GetOrCreateByUser(ctx context.Context, userID int64) (*wishlist.Wishlist, error)
	GetByUserWithItems(ctx context.Context, userID int64) (*wishlist.Wishlist, error)

	AddItem(ctx context.Context, userID int64, productID int64, variantID int64) error
	RemoveItem(ctx context.Context, userID int64, productID int64, variantID int64) error
	Clear(ctx context.Context, userID int64) error
}