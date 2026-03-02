package inventory

import "errors"

var (
	ErrLocationNotFound    = errors.New("inventory location not found")
	ErrItemNotFound        = errors.New("inventory item not found")
	ErrLocationCodeConflict = errors.New("location code already exists")
	ErrItemConflict        = errors.New("inventory item already exists for this variant and location")
	ErrInvalidName         = errors.New("location name is required and must be at most 150 characters")
	ErrInvalidLocationCode = errors.New("location code must be at most 50 characters")
	ErrInvalidQuantity     = errors.New("quantity cannot be negative")
	ErrInsufficientStock   = errors.New("insufficient available stock")
)