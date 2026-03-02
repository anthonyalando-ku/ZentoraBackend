package discount

import (
	"database/sql"
	"time"
)

type DiscountType string

const (
	TypePercentage DiscountType = "percentage"
	TypeFixed      DiscountType = "fixed"
)

type TargetType string

const (
	TargetProduct  TargetType = "product"
	TargetCategory TargetType = "category"
	TargetBrand    TargetType = "brand"
)

type Discount struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	Code            sql.NullString `json:"code"`
	DiscountType    DiscountType   `json:"discount_type"`
	Value           float64        `json:"value"`
	MinOrderAmount  sql.NullFloat64 `json:"min_order_amount"`
	MaxRedemptions  sql.NullInt64  `json:"max_redemptions"`
	StartsAt        sql.NullTime   `json:"starts_at"`
	EndsAt          sql.NullTime   `json:"ends_at"`
	IsActive        bool           `json:"is_active"`
	CreatedAt       time.Time      `json:"created_at"`
}

type DiscountTarget struct {
	DiscountID int64      `json:"discount_id"`
	TargetType TargetType `json:"target_type"`
	TargetID   int64      `json:"target_id"`
}

type DiscountRedemption struct {
	ID         int64         `json:"id"`
	DiscountID int64         `json:"discount_id"`
	OrderID    int64         `json:"order_id"`
	UserID     sql.NullInt64 `json:"user_id"`
	RedeemedAt time.Time     `json:"redeemed_at"`
}

type DiscountWithTargets struct {
	Discount
	Targets []DiscountTarget `json:"targets"`
}