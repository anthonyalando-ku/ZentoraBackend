package order

import "time"

type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending"
)

type Order struct {
	ID              int64
	UserID           *int64
	CartID           *int64
	OrderNumber      string
	Status           OrderStatus
	Subtotal         float64
	DiscountAmount   float64
	TaxAmount        float64
	ShippingFee      float64
	TotalAmount      float64
	Currency         string
	ShippingMethodID *int64

	Shipping ShippingInfo

	CreatedAt time.Time
	UpdatedAt time.Time

	Items []OrderItem
}

type OrderItem struct {
	ID             int64
	OrderID         int64
	ProductID       int64
	VariantID       *int64
	ProductName     string
	ProductSlug     *string
	VariantSKU      *string
	VariantName     *string
	ImageURL        *string
	UnitPrice       float64
	Quantity        int
	DiscountAmount  float64
	TaxRate         float64
	TotalPrice      float64
	Currency        string
}

type ShippingInfo struct {
	FullName      string  `json:"full_name"`
	Phone         string  `json:"phone"`
	Country       string  `json:"country"`
	County        *string `json:"county,omitempty"`
	City          string  `json:"city"`
	Area          *string `json:"area,omitempty"`
	PostalCode    *string `json:"postal_code,omitempty"`
	AddressLine1  string  `json:"address_line_1"`
	AddressLine2  *string `json:"address_line_2,omitempty"`
}