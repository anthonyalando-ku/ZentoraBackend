package order

import "errors"

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrOutOfStock         = errors.New("out of stock")
	ErrCartNotFound       = errors.New("cart not found")
	ErrAddressNotFound    = errors.New("address not found")
	ErrVariantNotFound    = errors.New("variant not found")
	ErrProductNotFound    = errors.New("product not found")
	ErrNotFound     = errors.New("not found")
)