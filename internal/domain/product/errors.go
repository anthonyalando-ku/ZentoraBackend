package product

import "errors"

var (
	ErrNotFound          = errors.New("product not found")
	ErrImageNotFound     = errors.New("product image not found")
	ErrSlugConflict      = errors.New("product slug already exists")
	ErrInvalidName       = errors.New("product name is required and must be at most 255 characters")
	ErrInvalidSlug       = errors.New("product slug is required and must be at most 255 characters")
	ErrInvalidPrice      = errors.New("base price must be greater than zero")
	ErrInvalidStatus     = errors.New("status must be 'active', 'draft', or 'archived'")
	ErrBrandRequired     = errors.New("brand_id is required")
	ErrCategoryRequired  = errors.New("at least one category is required")
	ErrImageRequired     = errors.New("at least one image is required")
	ErrVariantRequired   = errors.New("at least one variant is required")
	ErrInvalidSortOrder  = errors.New("sort_order must be >= 0")
)