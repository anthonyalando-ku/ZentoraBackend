package category

import "errors"

var (
	ErrNotFound       = errors.New("category not found")
	ErrSlugConflict   = errors.New("category slug already exists")
	ErrCircularParent = errors.New("category cannot be its own ancestor")
	ErrInvalidName    = errors.New("category name is required and must be at most 255 characters")
	ErrInvalidParent  = errors.New("parent_id must be a positive integer")
	ErrInvalidCategoryID = errors.New("all category_ids must be positive integers")
)