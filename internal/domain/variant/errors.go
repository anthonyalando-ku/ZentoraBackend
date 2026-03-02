package variant

import "errors"

var (
	ErrNotFound      = errors.New("variant not found")
	ErrSKUConflict   = errors.New("variant SKU already exists")
	ErrInvalidSKU    = errors.New("SKU is required and must be at most 100 characters")
	ErrInvalidPrice  = errors.New("price must be greater than zero")
	ErrInvalidWeight = errors.New("weight must be greater than zero")
)