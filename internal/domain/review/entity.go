package review

import "time"

type Review struct {
	ID                int64
	UserID            int64
	ProductID         int64
	OrderItemID       *int64
	Rating            int
	Comment           *string
	IsVerifiedPurchase bool
	CreatedAt         time.Time
}

type CreateRequest struct {
	ProductID   int64   `json:"product_id"`
	OrderItemID int64   `json:"order_item_id"`
	Rating      int     `json:"rating"`
	Comment     *string `json:"comment,omitempty"`
}