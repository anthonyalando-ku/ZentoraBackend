package wishlist

import "time"

type Wishlist struct {
	ID        int64
	UserID    int64
	CreatedAt time.Time
	Items     []WishlistItem
}

type WishlistItem struct {
	WishlistID int64
	ProductID  int64
	VariantID  int64
	AddedAt    time.Time
}