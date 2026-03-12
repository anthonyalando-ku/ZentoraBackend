package cart

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrCartNotFound      = errors.New("cart not found")
	ErrCartItemNotFound  = errors.New("cart item not found")
	ErrForbidden         = errors.New("forbidden")
	ErrVariantRequired   = errors.New("variant_id is required")
)