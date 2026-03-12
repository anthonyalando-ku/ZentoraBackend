package cart

import "time"

type CartStatus string

const (
	CartStatusActive    CartStatus = "active"
	CartStatusConverted CartStatus = "converted"
	CartStatusAbandoned CartStatus = "abandoned"
)

type Cart struct {
	ID        int64
	UserID    int64
	Status    CartStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Items     []CartItem
}

type CartItem struct {
	ID           int64
	CartID       int64
	ProductID    int64
	VariantID    int64
	Quantity     int
	PriceAtAdded string
	AddedAt      time.Time
}

type UpsertCartItemInput struct {
	ProductID    int64
	VariantID    int64
	Quantity     int
	PriceAtAdded string
}