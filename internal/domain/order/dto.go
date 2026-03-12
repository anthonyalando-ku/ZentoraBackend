package order


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
