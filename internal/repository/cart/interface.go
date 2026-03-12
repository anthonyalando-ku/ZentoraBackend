package cartrepo

import (
	"context"

	"zentora-service/internal/domain/cart"
)

type Repository interface {
	GetOrCreateActiveCartForUser(ctx context.Context, userID int64) (*cart.Cart, error)
	GetActiveCartWithItemsForUser(ctx context.Context, userID int64) (*cart.Cart, error)

	GetCartItems(ctx context.Context, cartID int64) ([]cart.CartItem, error)

	UpsertCartItem(ctx context.Context, cartID int64, in cart.UpsertCartItemInput) (*cart.CartItem, error)
	RemoveCartItem(ctx context.Context, cartID int64, itemID int64) error

	ClearCart(ctx context.Context, cartID int64) error
	MarkCartConverted(ctx context.Context, cartID int64) error
}