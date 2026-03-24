package order

import (
	"time"
)


type PaymentMethod string

const (
	PaymentPayOnDelivery PaymentMethod = "pay_on_delivery"
)

type CreateItem struct {
	ProductID int64 `json:"product_id"`
	VariantID int64 `json:"variant_id"`
	Quantity  int   `json:"quantity"`
}


type CreateGuestOrderRequest struct {
	Items          []CreateItem `json:"items"`
	Shipping       ShippingInfo `json:"shipping"`
	DeliveryMethod *string      `json:"delivery_method,omitempty"`
	PaymentMethod  *PaymentMethod `json:"payment_method,omitempty"`
}

type CreateUserOrderRequest struct {
	// Either CartID is set OR Items is non-empty.
	CartID         *int64       `json:"cart_id,omitempty"`
	Items          []CreateItem `json:"items,omitempty"`

	DeliveryMethod *string        `json:"delivery_method,omitempty"`
	PaymentMethod  *PaymentMethod `json:"payment_method,omitempty"`

	// Address selection: if nil, use default address
	AddressID *int64 `json:"address_id,omitempty"`
}


type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status"` // required
	Note   string      `json:"note"`   // optional (future: audit trail)
}

// For admin dashboard
type OrderStatsResponse struct {
	TotalOrders     int64   `json:"total_orders"`
	PendingOrders   int64   `json:"pending_orders"`
	CompletedOrders int64   `json:"completed_orders"`
	CancelledOrders int64   `json:"cancelled_orders"`

	RevenueTotal   float64 `json:"revenue_total"`
	RevenueToday   float64 `json:"revenue_today"`
	Revenue7Days   float64 `json:"revenue_7_days"`
	OrdersToday    int64   `json:"orders_today"`
	Orders7Days    int64   `json:"orders_7_days"`

	UpdatedAt time.Time `json:"updated_at"`
}